package main

import (
	"github.com/c0dered273/automation-remote-controller/internal/common/loggers"
	"github.com/c0dered273/automation-remote-controller/internal/common/validators"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/configs"
	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	configFileName = "tgbot_config"
	configFilePath = []string{
		".",
		"./configs/",
	}
)

func main() {
	initLogger := loggers.NewDefaultLogger()
	initLogger.Info().Msg("remote-control-tg-bot: init")
	validator := validators.NewValidatorTagName("mapstructure")
	cfg, err := configs.NewTGBotConfiguration(configFileName, configFilePath, initLogger, validator)
	if err != nil {
		initLogger.Fatal().Err(err).Msg("remote-control-tg-bot: config init failed")
	}
	logger := loggers.NewLogger(cfg.Logger, "remote-control-tg-bot")

	// tg bot
	bot, err := tg.NewBotAPI(cfg.BotToken)
	if err != nil {
		logger.Fatal().Err(err).Msg("remote-control-tg-bot: bot init error")
	}
	bot.Debug = true

	logger.Info().Msgf("remote-control-tg-bot: authorized on account: %s", bot.Self.UserName)

	u := tg.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// buttons
	keyboard := tg.NewReplyKeyboard(
		tg.NewKeyboardButtonRow(
			tg.NewKeyboardButton("ON"),
			tg.NewKeyboardButton("OFF"),
		),
	)

	inlineMainMenu := tg.NewInlineKeyboardMarkup(
		tg.NewInlineKeyboardRow(
			tg.NewInlineKeyboardButtonData("Лампочка 1", "switchLamp01"),
		),
	)

	for update := range updates {
		if update.Message != nil {
			logger.Info().Msgf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := tg.NewMessage(update.Message.Chat.ID, update.Message.Text)
			//msg.ReplyToMessageID = update.Message.MessageID

			switch update.Message.Text {
			case "/menu":
				msg.Text = "Main menu"
				msg.ReplyMarkup = inlineMainMenu
			case "/start":
				msg.ReplyMarkup = keyboard
			case "/stop":
				msg.ReplyMarkup = tg.NewRemoveKeyboard(true)
			}

			if _, err = bot.Send(msg); err != nil {
				logger.Fatal().Err(err).Send()
			}
		} else if update.CallbackQuery != nil {
			callback := tg.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
			if _, err = bot.Request(callback); err != nil {
				logger.Fatal().Err(err).Send()
			}

			msg := tg.NewMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Data)
			if _, err = bot.Send(msg); err != nil {
				logger.Fatal().Err(err).Send()
			}
		}
	}
}
