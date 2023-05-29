package services

import (
	"container/list"
	"context"
	"encoding/json"

	"github.com/c0dered273/automation-remote-controller/internal/common/model"
	"github.com/c0dered273/automation-remote-controller/internal/common/proto"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	chatID int64 = 251967291
)

type EventMultiService struct {
	proto.UnimplementedEventMultiServiceServer
	ctx            context.Context
	logger         zerolog.Logger
	eventQueue     *list.List
	notifyCallback func(chatID int64, text string) error
}

func (s *EventMultiService) EventStreaming(srv proto.EventMultiService_EventStreamingServer) error {
	recvChan := make(chan *model.Event)
	sendChan := make(chan *model.Event)

	go func() {
		for {
			select {
			default:
			case <-s.ctx.Done():
				return
			}

			e := &model.Event{}
			req, err := srv.Recv()
			if err != nil {
				e.Err = err
				s.logger.Error().Err(err)
				continue
			}
			s.logger.Warn().Msg("RECV")
			e.Event = req
			recvChan <- e
			s.logger.Info().Str("action", req.Action.String()).RawJSON("payload", req.Payload).Msg("incoming event")
		}
	}()

	go func() {
		for {
			select {
			default:
			case <-s.ctx.Done():
				return
			}

			e := <-sendChan
			err := srv.Send(e.Event)
			if err != nil {
				s.logger.Error().Err(err)
			}
			s.logger.Info().Str("action", e.Event.Action.String()).RawJSON("payload", e.Event.Payload).Msg("send event")
			s.logger.Warn().Msg("SEND")
		}
	}()

	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		case e := <-recvChan:
			// RECV
			switch e.Event.Action {
			case proto.Action_NOTIFICATION:
				var notifyEvent model.NotifyEvent
				err := json.Unmarshal(e.Event.Payload, &notifyEvent)
				if err != nil {
					s.logger.Error().Err(err)
					return status.Errorf(codes.Internal, "Internal error")
				}

				err = s.notifyCallback(chatID, notifyEvent.Text)
				if err != nil {
					s.logger.Error().Err(err)
					return status.Errorf(codes.Internal, "Internal error")
				}
			}
		default:
			// SEND
			e := s.eventQueue.Front()
			if e != nil {
				s.eventQueue.Remove(e)
				s.logger.Warn().Msg("FROM QUEUE")
				payload, err := json.Marshal(&e.Value)
				if err != nil {
					return status.Errorf(codes.Internal, "Internal error")
				}
				event := &model.Event{
					Event: &proto.Event{
						Id:      uuid.NewString(),
						Action:  proto.Action_SWITCH,
						Payload: payload,
					},
					Err: err,
				}
				sendChan <- event
			}
		}
	}
}

func NewEventMultiService(ctx context.Context, logger zerolog.Logger, eventQueue *list.List, notifyCallback func(chatID int64, text string) error) *EventMultiService {
	return &EventMultiService{
		ctx:            ctx,
		logger:         logger,
		eventQueue:     eventQueue,
		notifyCallback: notifyCallback,
	}
}
