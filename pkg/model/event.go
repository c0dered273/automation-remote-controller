package model

import (
	"fmt"
	"strings"
)

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

func NewAction(s string) (Action, error) {
	for i, a := range actions {
		if strings.EqualFold(a, s) {
			return Action(i), nil
		}
	}
	return 0, fmt.Errorf("actions: failed to parse action <%s>", s)
}

type NotifyEvent struct {
	Text string `json:"text"`
}

type ActionEvent struct {
	DeviceID string `json:"device_id"`
	Action   Action `json:"action"`
}
