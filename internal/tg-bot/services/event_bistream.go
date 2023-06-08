package services

import (
	"context"
	"errors"

	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/model"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/repository"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/users"
	"github.com/c0dered273/automation-remote-controller/pkg/collections"
	"github.com/c0dered273/automation-remote-controller/pkg/proto"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// EventMultiService обрабатывает соединения от клиентских приложений
type EventMultiService struct {
	proto.UnimplementedEventMultiServiceServer
	ctx         context.Context
	logger      zerolog.Logger
	clients     *collections.ConcurrentMap[string, *model.ClientEvents]
	notify      chan<- model.Notification
	userService users.UserService
}

// EventStreaming получает двунаправленный поток отк клиента, достает из метаданных идентификаторы, идентифицирует клиента.
// В случае валидного клиента создает структуру ClientEvents, добавляет ее в общий словарь,
// чтобы обработчики команд от telegram могли обратиться к конкретному клиентскому приложению.
// Также запускается циклический опрос сообщений от клиентского приложения и перенаправление сообщений в каналы,
// привязанные к конкретному пользователю.
// Идентификация пользователя происходит в 2 этап:
// 1. При установке соединения проверяется валидность сертификата пользователя через tls handshake
// 2. Проверяется существование пользователя с указанным именем и идентификатором сертификата
func (s *EventMultiService) EventStreaming(stream proto.EventMultiService_EventStreamingServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		s.logger.Error().Msg("event streaming: failed to get stream metadata")
		return status.Error(codes.InvalidArgument, "failed to get stream metadata")
	}

	var tgName string
	values := md.Get("X-Username")
	if len(values) == 0 {
		s.logger.Error().Msg("event streaming: failed to get client username")
		return status.Error(codes.InvalidArgument, "failed to get client username")
	}
	tgName = values[0]
	var certID string
	values = md.Get("X-ClientID")
	if len(values) == 0 {
		s.logger.Error().Msg("event streaming: failed to get client id")
		return status.Error(codes.InvalidArgument, "failed to get client id")
	}
	certID = values[0]

	user, err := s.userService.FindUserByClientID(s.ctx, certID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return status.Error(codes.Unauthenticated, "client cert id is invalid")
		}
		return status.Error(codes.Internal, "Internal error")
	}
	if tgName != user.TGUser {
		return status.Error(codes.Unauthenticated, "username is invalid")
	}
	if user.ChatID == 0 {
		return status.Error(codes.InvalidArgument, "unable to get chatID, please register with telegram")
	}

	clientEvents := model.NewClientEvents(s.ctx, user.ChatID, s.notify, user.NotifyEnabled, s.logger)
	clientEvents.ContinuousReadAndNotify()
	s.clients.Put(tgName, clientEvents)
	s.logger.Info().Msgf("new connect from %s, %s", tgName, certID)

	// Получаем события из стрима. Метод stream.Recv() блокирующий, поэтому запускаем в отдельной горутине
	go func() {
		for {
			select {
			default:
			case <-s.ctx.Done():
				clientEvents.Err <- s.ctx.Err()
				return
			}
			e := model.Event{}
			recv, err := stream.Recv()
			if err != nil {
				clientEvents.Err <- err
				return
			}
			if recv == nil {
				clientEvents.Err <- errors.New("failed to receive event - connection lost")
				return
			}

			e.E = recv
			clientEvents.Recv <- &e
			s.logger.Info().Str("action", e.E.Action.String()).RawJSON("payload", e.E.Payload).Msgf("incoming event from %s", tgName)
		}
	}()

	for {
		select {
		case <-s.ctx.Done():
			return nil
		case err := <-clientEvents.Err:
			return status.Errorf(codes.Internal, "%v", err)
		case e := <-clientEvents.Send:
			err := stream.Send(e.E)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to send event, %v", err)
			}
		}
	}

}

// NewEventMultiService создает сервис обслуживания клиентских приложений
func NewEventMultiService(
	ctx context.Context,
	logger zerolog.Logger,
	clients *collections.ConcurrentMap[string, *model.ClientEvents],
	notify chan<- model.Notification,
	userService users.UserService,
) *EventMultiService {
	return &EventMultiService{
		ctx:         ctx,
		logger:      logger,
		clients:     clients,
		notify:      notify,
		userService: userService,
	}
}
