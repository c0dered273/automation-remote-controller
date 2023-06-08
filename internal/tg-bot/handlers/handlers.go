package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/model"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/users"
	"github.com/c0dered273/automation-remote-controller/pkg/collections"
	pkgmodel "github.com/c0dered273/automation-remote-controller/pkg/model"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"
)

const (
	PositiveCheck = "\u2705"
	NegativeCross = "\u274C"
)

// @Description Обработчики команд от telegram api

// MenuHandler /menu - главное меню
func MenuHandler(ctx context.Context, logger zerolog.Logger, userService users.UserService) func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	return func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
		var chatID int64
		var username string
		if update.Message == nil {
			chatID = update.CallbackQuery.Message.Chat.ID
			username = update.CallbackQuery.From.UserName
		} else {
			chatID = update.Message.Chat.ID
			username = update.Message.From.UserName
		}

		if prevMsg, ok := userService.GetUserLastMessage(username); ok {
			delMsg := tgbotapi.NewDeleteMessage(chatID, prevMsg.MessageID)
			_, _ = botApi.Send(delMsg)
		}

		msg := tgbotapi.NewMessage(chatID, "Error: unknown")
		if err := userService.SetUserChatID(ctx, username, chatID); err != nil {
			logger.Error().Err(err).Send()
			msg.Text = "Error: unknown user"
		} else {
			msg.Text = "Главное меню"
			inlineMainMenu := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Состояние", "handler:status"),
					tgbotapi.NewInlineKeyboardButtonData("Освещение", "handler:lightControl"),
				),
			)
			msg.ReplyMarkup = inlineMainMenu
		}

		sent, err := botApi.Send(msg)
		if err != nil {
			logger.Fatal().Err(err).Send()
		}
		userService.SetUserLastMessage(username, sent)
	}
}

// StartNotificationsHandler /start - включить уведомления
func StartNotificationsHandler(ctx context.Context, logger zerolog.Logger, userService users.UserService, clients *collections.ConcurrentMap[string, *model.ClientEvents]) func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	return func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
		username := update.Message.From.UserName
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Error: unknown")
		if err := userService.SetNotification(ctx, username, true); err != nil {
			logger.Error().Err(err).Send()
			msg.Text = "Error: unknown user"
		} else {
			client, ok := clients.Get(username)
			if !ok {
				logger.Error().Msg("handler: failed to find client grpc stream")
			}
			client.IsNotify = true

			msg.Text = "notifications enabled"
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		}

		if _, err := botApi.Send(msg); err != nil {
			logger.Fatal().Err(err).Send()
		}
	}
}

// StopNotificationsHandler /stop - отключить уведомления
func StopNotificationsHandler(ctx context.Context, logger zerolog.Logger, userService users.UserService, clients *collections.ConcurrentMap[string, *model.ClientEvents]) func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	return func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
		username := update.Message.From.UserName
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Error: unknown")
		if err := userService.SetNotification(ctx, username, false); err != nil {
			logger.Error().Err(err).Send()
			msg.Text = "Error: unknown user"
		} else {
			client, ok := clients.Get(username)
			if !ok {
				logger.Error().Msg("handler: failed to find client grpc stream")
			}
			client.IsNotify = false

			msg.Text = "notifications disabled"
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		}

		if _, err := botApi.Send(msg); err != nil {
			logger.Fatal().Err(err).Send()
		}
	}
}

// StatusHandler :status - состояние систем
func StatusHandler(ctx context.Context, logger zerolog.Logger, userService users.UserService) func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	return func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
		if _, err := botApi.Request(callback); err != nil {
			logger.Fatal().Err(err).Send()
		}

		username := update.CallbackQuery.From.UserName
		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Error: unknown")
		if userService.IsUserExists(ctx, username) {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("%s Электричество\n", PositiveCheck))
			sb.WriteString(fmt.Sprintf("%s Отопление\n", PositiveCheck))
			sb.WriteString(fmt.Sprintf("%s Водоснабжение\n", NegativeCross))
			sb.WriteString(fmt.Sprintf("%s Вентиляция\n", PositiveCheck))
			msg.Text = sb.String()
		} else {
			msg.Text = "Error: unknown user"
		}
		sent, err := botApi.Send(msg)
		if err != nil {
			logger.Fatal().Err(err).Send()
		}
		userService.SetUserLastMessage(username, sent)
	}
}

// LightControlHandler :lightControl - меню управления освещением
func LightControlHandler(ctx context.Context, logger zerolog.Logger, userService users.UserService) func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	return func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
		if _, err := botApi.Request(callback); err != nil {
			logger.Fatal().Err(err).Send()
		}

		username := update.CallbackQuery.From.UserName
		if prevMsg, ok := userService.GetUserLastMessage(username); ok {
			delMsg := tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, prevMsg.MessageID)
			_, _ = botApi.Send(delMsg)
		}

		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Error: unknown")
		if userService.IsUserExists(ctx, username) {
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
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Главное меню", "/menu"),
				),
			)
			msg.Text = "Освещение"
			msg.ReplyMarkup = inlineButtons
		} else {
			msg.Text = "Error: unknown user"
		}

		sent, err := botApi.Send(msg)
		if err != nil {
			logger.Fatal().Err(err).Send()
		}
		userService.SetUserLastMessage(username, sent)
	}
}

// LampMenuHandler :lampMenu - меню управления лампой
// параметр lampID - идентификатор устройства, для которого нужно вывести меню
func LampMenuHandler(ctx context.Context, logger zerolog.Logger, userService users.UserService) func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	return func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
		if _, err := botApi.Request(callback); err != nil {
			logger.Fatal().Err(err).Send()
		}

		username := update.CallbackQuery.From.UserName
		if prevMsg, ok := userService.GetUserLastMessage(username); ok {
			delMsg := tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, prevMsg.MessageID)
			_, _ = botApi.Send(delMsg)
		}

		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Error: unknown")
		if userService.IsUserExists(ctx, username) {
			reqParams := ParseReqParams(update.CallbackQuery.Data)
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

			msg.Text = sb.String()
			msg.ReplyMarkup = inlineButtons
		} else {
			msg.Text = "Error: unknown user"
		}
		sent, err := botApi.Send(msg)
		if err != nil {
			logger.Fatal().Err(err).Send()
		}
		userService.SetUserLastMessage(username, sent)
	}
}

// LampSwitchHandler :lampSwitch - выполнение указанной команды для устройства
// параметр lampID - идентификатор устройства, для которого нужно выполнить команду
// параметр action - команды
func LampSwitchHandler(ctx context.Context, logger zerolog.Logger, userService users.UserService, clients *collections.ConcurrentMap[string, *model.ClientEvents]) func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	return func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
		if _, err := botApi.Request(callback); err != nil {
			logger.Fatal().Err(err).Send()
		}

		username := update.CallbackQuery.From.UserName
		if prevMsg, ok := userService.GetUserLastMessage(username); ok {
			delMsg := tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, prevMsg.MessageID)
			_, _ = botApi.Send(delMsg)
		}

		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Error: unknown")
		if userService.IsUserExists(ctx, username) {
			reqParams := ParseReqParams(update.CallbackQuery.Data)
			lampID := reqParams["lampID"][0]
			action := reqParams["action"][0]

			eventAction, _ := pkgmodel.NewAction(action)
			event := pkgmodel.ActionEvent{
				DeviceID: lampID,
				Action:   eventAction,
			}

			client, ok := clients.Get(update.CallbackQuery.From.UserName)
			if !ok {
				logger.Error().Msg("handler: failed to find client grpc stream")
			}

			err := client.SendAction(event)
			if err != nil {
				logger.Error().Err(err).Msg("handler: failed to send action")
			}

			var sb strings.Builder
			sb.WriteString("\xF0\x9F\x92\xA1")
			sb.WriteString(fmt.Sprintf("%s\n", lampID))
			sb.WriteString(fmt.Sprintf("%s\n", action))

			inlineButtons := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Назад", fmt.Sprintf("handler:lampMenu?lampID=%s", lampID)),
				),
			)
			msg.Text = sb.String()
			msg.ReplyMarkup = inlineButtons

		} else {
			msg.Text = "Error: unknown user"
		}
		sent, err := botApi.Send(msg)
		if err != nil {
			logger.Fatal().Err(err).Send()
		}
		userService.SetUserLastMessage(username, sent)
	}
}
