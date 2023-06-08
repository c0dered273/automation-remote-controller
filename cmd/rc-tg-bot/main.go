package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/model"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/server"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/services"
	"github.com/c0dered273/automation-remote-controller/pkg/collections"
	"github.com/c0dered273/automation-remote-controller/pkg/loggers"
)

//	@Title			rc-tg-bot
//	@Description	Приложение позволяет передавать команды и получать сообщения пользователю telegram с одной стороны и клиентскому приложению с другой
//	@Version		0.0.1

func main() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	config := server.ReadConfig()
	logger := loggers.NewLogger(server.LogWriter, config.Logger, "remote-control-tg-bot")
	s := services.NewServices(config.DatabaseUri, logger)
	clientsMap := collections.NewConcurrentMap[string, *model.ClientEvents]()

	// TG
	bot := server.NewBotServer(serverCtx, config.BotToken, s, clientsMap, logger)
	bot.ServeAndNotify()
	botNotify := bot.GetNotifyChan()

	// gRPC
	listen, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		logger.Fatal().Err(err).Send()
	}
	grpcServer, err := server.NewGRPCServer(serverCtx, config, logger, clientsMap, botNotify, s.UserService)
	if err != nil {
		logger.Fatal().Err(err).Msg("remote-control-tg-bot: server init error")
	}

	go func() {
		logger.Info().Msgf("gRPC server started at %v", config.Port)
		err = grpcServer.Serve(listen)
		if err != nil {
			logger.Fatal().Err(err).Send()
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
