package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/c0dered273/automation-remote-controller/internal/common/loggers"
	"github.com/c0dered273/automation-remote-controller/internal/common/validators"
	"github.com/c0dered273/automation-remote-controller/internal/user-account/server"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//	@Title			user-account-api
//	@Description	Сервис управления личным кабинетом пользователя.
//	@Version		0.0.1

var (
	shutdownTimeout = 25 * time.Second
)

func main() {
	serverCtx, serverStopCtx := context.WithCancel(context.Background())
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	config := server.ReadConfig()
	logger := loggers.NewLogger(server.LogWriter, config.Logger, "user-account-api")
	validator := validators.NewValidatorWithTagFieldName("json", logger)
	services, db := server.NewServices(config, logger)
	e := server.NewEchoServer(services, config, logger, validator)

	go func() {
		if err := e.Start(":" + config.Port); err != nil {
			if err == http.ErrServerClosed {
				return
			}
			logger.Fatal().Err(err).Send()
		}
	}()

	go func() {
		<-shutdown
		shutdownCtx, shutdownCancelCtx := context.WithTimeout(serverCtx, shutdownTimeout)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				logger.Fatal().Msg("server: graceful shutdown timed out.. forcing exit.")
			}
		}()

		if err := e.Shutdown(shutdownCtx); err != nil {
			logger.Fatal().Err(err).Msg("server: graceful shutdown failed")
		}
		err := db.Close()
		if err != nil {
			logger.Fatal().Err(err).Send()
		}

		logger.Info().Msg("server: shutting down")
		serverStopCtx()
		shutdownCancelCtx()
	}()

	<-serverCtx.Done()
}
