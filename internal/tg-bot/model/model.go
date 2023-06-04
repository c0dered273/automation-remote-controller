package model

import "github.com/c0dered273/automation-remote-controller/pkg/proto"

// Event внутреннее описание события
type Event struct {
	E   *proto.Event
	Err error
}

// Notification внутреннее описание события уведомления
type Notification struct {
	ChatID int64
	Text   string
}

func NewNotification(chatID int64, text string) Notification {
	return Notification{
		ChatID: chatID,
		Text:   text,
	}
}
