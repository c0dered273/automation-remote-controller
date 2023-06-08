package model

import (
	"context"
	"encoding/json"
	"fmt"

	pkgmodel "github.com/c0dered273/automation-remote-controller/pkg/model"
	"github.com/c0dered273/automation-remote-controller/pkg/proto"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ClientEvents обеспечивает связь между пользователем telegram и конкретным клиентским приложением
// структура содержит каналы, которые привязаны к стриму подключенного клиентского приложения
type ClientEvents struct {
	ctx context.Context
	// Recv прием сообщений от клиентского приложения
	Recv chan *Event
	// Send отправка сообщений в клиентское приложение
	Send chan *Event
	// Err обработка ошибок, при появлении в канале объекта, клиентский стрим закрывается
	Err chan error
	// chatID идентификатор чата telegram, в который будут отправлены уведомления
	chatID int64
	// botNotify канал отправки сообщений непосредственно в чат пользователю
	botNotify chan<- Notification
	// IsNotify флаг показывает отправлять ли пользователю сообщения
	IsNotify bool
	logger   zerolog.Logger
}

// SendAction отправить событие клиентскому приложению
func (e *ClientEvents) SendAction(a pkgmodel.ActionEvent) error {
	payload, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("client events: failed to marshal action, %w", err)
	}
	event := Event{
		E: &proto.Event{
			Id:      uuid.NewString(),
			Action:  proto.Action_SWITCH,
			Payload: payload,
		},
		Err: nil,
	}
	e.Send <- &event
	return nil
}

// ContinuousReadAndNotify ожидает событие от клиентского приложения и передает его непосредственно в чат пользователю
func (e *ClientEvents) ContinuousReadAndNotify() {
	go func() {
		for {
			select {
			case <-e.ctx.Done():
				e.Err <- e.ctx.Err()
				return
			case recv := <-e.Recv:
				if !e.IsNotify {
					continue
				}
				switch recv.E.Action {
				case proto.Action_NOTIFICATION:
					var notifyEvent pkgmodel.NotifyEvent
					err := json.Unmarshal(recv.E.Payload, &notifyEvent)
					if err != nil {
						e.Err <- fmt.Errorf("client events: failed unmarshal event, %w", err)
						return
					}
					e.botNotify <- NewNotification(e.chatID, notifyEvent.Text)
				}
			}
		}
	}()
}

// NewClientEvents создает настроенную структуру ClientEvents
func NewClientEvents(ctx context.Context, chatID int64, botNotify chan<- Notification, isNotify bool, logger zerolog.Logger) *ClientEvents {
	return &ClientEvents{
		ctx:       ctx,
		Recv:      make(chan *Event),
		Send:      make(chan *Event),
		Err:       make(chan error),
		chatID:    chatID,
		botNotify: botNotify,
		IsNotify:  isNotify,
		logger:    logger,
	}
}
