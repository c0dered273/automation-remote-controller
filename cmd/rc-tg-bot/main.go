package main

import (
	"container/list"
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/c0dered273/automation-remote-controller/internal/common/loggers"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/handlers"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/server"
)

func main() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	config := server.ReadConfig()
	logger := loggers.NewLogger(server.LogWriter, config.Logger, "remote-control-tg-bot")
	//validator := validators.NewValidatorWithTagFieldName("mapstructure", logger)
	//db, err := storage.NewConnection(config.DatabaseUri)
	//if err != nil {
	//	logger.Fatal().Err(err)
	//}
	//repo := users.NewRepo(db)
	eventQueue := list.New()

	// tg bot
	h := server.NewMessageHandler(logger)
	h.Message("/menu", handlers.MenuHandler(logger))
	h.Message("/start", handlers.StartNotificationsHandler(logger))
	h.Message("/stop", handlers.StopNotificationsHandler(logger))
	h.Callback("status", handlers.StatusHandler(logger))
	h.Callback("lightControl", handlers.LightControlHandler(logger))
	h.Callback("lampMenu", handlers.LampMenuHandler(logger))
	h.Callback("lampSwitch", handlers.LampSwitchHandler(logger, eventQueue))

	bot, err := server.NewTGBot(config, logger, h)
	if err != nil {
		logger.Fatal().Err(err).Msg("remote-control-tg-bot: bot init error")
	}

	go func() {
		bot.Serve()
	}()

	// gRPC
	listen, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		logger.Fatal().Err(err)
	}
	grpcServer, err := server.NewGRPCServer(serverCtx, config, logger, bot, eventQueue)
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
}
