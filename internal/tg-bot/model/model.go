package model

import "github.com/c0dered273/automation-remote-controller/pkg/proto"

type Event struct {
	E   *proto.Event
	Err error
}

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
