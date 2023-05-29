package handlers

import (
	"container/list"
	"fmt"
	"strings"

	"github.com/c0dered273/automation-remote-controller/internal/common/model"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/server"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"
)

const (
	PositiveCheck = "\u2705"
	NegativeCross = "\u274C"
)

func MenuHandler(logger zerolog.Logger) func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	return func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
		inlineMainMenu := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Состояние", "handler:status"),
				tgbotapi.NewInlineKeyboardButtonData("Освещение", "handler:lightControl"),
			),
		)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Главное меню")
		msg.ReplyMarkup = inlineMainMenu
		if _, err := botApi.Send(msg); err != nil {
			logger.Fatal().Err(err).Send()
		}
	}
}

func StartNotificationsHandler(logger zerolog.Logger) func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	return func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "bot notifications enabled")
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		if _, err := botApi.Send(msg); err != nil {
			logger.Fatal().Err(err).Send()
		}
	}
}

func StopNotificationsHandler(logger zerolog.Logger) func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	return func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "bot notifications disabled")
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		if _, err := botApi.Send(msg); err != nil {
			logger.Fatal().Err(err).Send()
		}
	}
}

func StatusHandler(logger zerolog.Logger) func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	return func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
		if _, err := botApi.Request(callback); err != nil {
			logger.Fatal().Err(err).Send()
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s Электричество\n", PositiveCheck))
		sb.WriteString(fmt.Sprintf("%s Отопление\n", PositiveCheck))
		sb.WriteString(fmt.Sprintf("%s Водоснабжение\n", NegativeCross))
		sb.WriteString(fmt.Sprintf("%s Вентиляция\n", PositiveCheck))

		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, sb.String())
		if _, err := botApi.Send(msg); err != nil {
			logger.Fatal().Err(err).Send()
		}
	}
}

func LightControlHandler(logger zerolog.Logger) func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	return func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
		if _, err := botApi.Request(callback); err != nil {
			logger.Fatal().Err(err).Send()
		}

		lamp01 := "Lamp001"
		lamp02 := "Lamp002"
		lamp03 := "Lamp003"

		inlineButtons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("\xF0\x9F\x92\xA1%s", lamp01), fmt.Sprintf("handler:lampMenu?lampID=%s", lamp01)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("\xF0\x9F\x92\xA1%s", lamp02), fmt.Sprintf("handler:lampMenu?lampID=%s", lamp02)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("\xF0\x9F\x92\xA1%s", lamp03), fmt.Sprintf("handler:lampMenu?lampID=%s", lamp03)),
			),
		)

		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Освещение")
		msg.ReplyMarkup = inlineButtons
		if _, err := botApi.Send(msg); err != nil {
			logger.Fatal().Err(err).Send()
		}
	}
}

func LampMenuHandler(logger zerolog.Logger) func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	return func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
		if _, err := botApi.Request(callback); err != nil {
			logger.Fatal().Err(err).Send()
		}

		reqParams := server.ParseReqParams(update.CallbackQuery.Data)
		lampID := reqParams["lampID"][0]

		var sb strings.Builder
		sb.WriteString("\xF0\x9F\x92\xA1")
		sb.WriteString(fmt.Sprintf("%s\n", lampID))

		inlineButtons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Включить", fmt.Sprintf("handler:lampSwitch?lampID=%s&action=switchON", lampID)),
				tgbotapi.NewInlineKeyboardButtonData("Отключить", fmt.Sprintf("handler:lampSwitch?lampID=%s&action=switchOFF", lampID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Назад", "handler:lightControl"),
			),
		)

		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, sb.String())
		msg.ReplyMarkup = inlineButtons
		if _, err := botApi.Send(msg); err != nil {
			logger.Fatal().Err(err).Send()
		}
	}
}

func LampSwitchHandler(logger zerolog.Logger, eventQueue *list.List) func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	return func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
		if _, err := botApi.Request(callback); err != nil {
			logger.Fatal().Err(err).Send()
		}

		reqParams := server.ParseReqParams(update.CallbackQuery.Data)
		lampID := reqParams["lampID"][0]
		action := reqParams["action"][0]

		eventAction, _ := model.NewAction(action)
		event := model.ActionEvent{
			DeviceID: lampID,
			Action:   eventAction,
		}

		logger.Warn().Msg("TO QUEUE")
		eventQueue.PushBack(event)

		var sb strings.Builder
		sb.WriteString("\xF0\x9F\x92\xA1")
		sb.WriteString(fmt.Sprintf("%s\n", lampID))
		sb.WriteString(fmt.Sprintf("%s\n", action))

		inlineButtons := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Назад", fmt.Sprintf("handler:lampMenu?lampID=%s", lampID)),
			),
		)
		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, sb.String())
		msg.ReplyMarkup = inlineButtons
		if _, err := botApi.Send(msg); err != nil {
			logger.Fatal().Err(err).Send()
		}
	}
}
