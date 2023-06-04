package model

import (
	"fmt"
	"strings"
)

// Action список команд для исполнительного устройства
type Action uint8

const (
	Empty Action = iota
	SwitchON
	SwitchOFF
	Toggle
)

var actions = []string{
	"Empty",
	"SwitchON",
	"SwitchOFF",
	"Toggle",
}

func (t Action) String() string {
	return actions[t]
}

// NewAction создает новый action из строки
func NewAction(s string) (Action, error) {
	for i, a := range actions {
		if strings.EqualFold(a, s) {
			return Action(i), nil
		}
	}
	return 0, fmt.Errorf("actions: failed to parse action <%s>", s)
}

// NotifyEvent payload для события уведомления
type NotifyEvent struct {
	Text string `json:"text"`
}

// ActionEvent payload для события действия
type ActionEvent struct {
	DeviceID string `json:"device_id"`
	Action   Action `json:"action"`
}
