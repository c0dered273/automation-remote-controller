package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/c0dered273/automation-remote-controller/internal/common/proto"
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
	return 0, errors.New(fmt.Sprintf("actions: failed to parse action <%s>", s))
}

type Event struct {
	Event *proto.Event
	Err   error
}

type NotifyEvent struct {
	TGName string `json:"tg_name"`
	Text   string `json:"text"`
}

type ActionEvent struct {
	DeviceID string `json:"device_id"`
	Action   Action `json:"action"`
}
