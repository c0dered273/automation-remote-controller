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

type ClientEvents struct {
	ctx       context.Context
	Recv      chan *Event
	Send      chan *Event
	Err       chan error
	chatID    int64
	botNotify chan<- Notification
	IsNotify  bool
	logger    zerolog.Logger
}

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

func (e *ClientEvents) ContinuousReadAndNotify() {
	go func() {
		for {
			select {
			case <-e.ctx.Done():
				e.Err <- e.ctx.Err()
				return
			case recv := <-e.Recv:
				if e.IsNotify {
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
		}
	}()
}

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
