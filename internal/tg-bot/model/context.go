package model

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type ChatContext struct {
	LastRequest  tgbotapi.Update        `json:"last_request"`
	LastResponse tgbotapi.MessageConfig `json:"last_response"`
	Properties   map[string]string      `json:"properties"`
}
