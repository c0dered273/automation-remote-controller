package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/c0dered273/automation-remote-controller/internal/common/loggers"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/server"
)

func main() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	config := server.ReadConfig()
	logger := loggers.NewLogger(server.LogWriter, config.Logger, "remote-control-tg-bot")
	//validator := validators.NewValidatorWithTagFieldName("mapstructure", logger)

	// gRPC
	listen, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		logger.Fatal().Err(err)
	}
	grpcServer, err := server.NewGRPCServer(config, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("remote-control-tg-bot: server init error")
	}

	go func() {
		logger.Info().Msgf("gRPC server started at %v", config.Port)
		err = grpcServer.Serve(listen)
		if err != nil {
			logger.Fatal().Err(err)
		}
	}()

	go func() {
		<-shutdown
		shutdownCtx, shutdownCancelCtx := context.WithTimeout(serverCtx, 30*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				logger.Fatal().Msg("server: graceful shutdown timed out.. forcing exit.")
			}
		}()

		grpcServer.GracefulStop()

		serverStopCtx()
		shutdownCancelCtx()
	}()

	<-serverCtx.Done()
	logger.Info().Msg("Server shutting down")

	//// tg bot
	//bot, err := tg.NewBotAPI(config.BotToken)
	//if err != nil {
	//	logger.Fatal().Err(err).Msg("remote-control-tg-bot: bot init error")
	//}
	//bot.Debug = true
	//
	//logger.Info().Msgf("remote-control-tg-bot: authorized on account: %s", bot.Self.UserName)
	//
	//u := tg.NewUpdate(0)
	//u.Timeout = 60
	//updates := bot.GetUpdatesChan(u)
	//
	//// buttons
	//keyboard := tg.NewReplyKeyboard(
	//	tg.NewKeyboardButtonRow(
	//		tg.NewKeyboardButton("ON"),
	//		tg.NewKeyboardButton("OFF"),
	//	),
	//)
	//
	//inlineMainMenu := tg.NewInlineKeyboardMarkup(
	//	tg.NewInlineKeyboardRow(
	//		tg.NewInlineKeyboardButtonData("Лампочка 1", "switchLamp01"),
	//	),
	//)
	//
	//for update := range updates {
	//	if update.Message != nil {
	//		logger.Info().Msgf("[%s] %s", update.Message.From.UserName, update.Message.Text)
	//
	//		msg := tg.NewMessage(update.Message.Chat.ID, update.Message.Text)
	//		//msg.ReplyToMessageID = update.Message.MessageID
	//
	//		switch update.Message.Text {
	//		case "/menu":
	//			msg.Text = "Main menu"
	//			msg.ReplyMarkup = inlineMainMenu
	//		case "/start":
	//			msg.ReplyMarkup = keyboard
	//		case "/stop":
	//			msg.ReplyMarkup = tg.NewRemoveKeyboard(true)
	//		}
	//
	//		if _, err = bot.Send(msg); err != nil {
	//			logger.Fatal().Err(err).Send()
	//		}
	//	} else if update.CallbackQuery != nil {
	//		callback := tg.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
	//		if _, err = bot.Request(callback); err != nil {
	//			logger.Fatal().Err(err).Send()
	//		}
	//
	//		msg := tg.NewMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Data)
	//		if _, err = bot.Send(msg); err != nil {
	//			logger.Fatal().Err(err).Send()
	//		}
	//	}
	//}
}
