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

func main() {
	config := server.ReadConfig()
	logger := loggers.NewLogger(server.LogWriter, config.Logger, "user-account-api")
	validator := validators.NewValidatorWithTagFieldName("json", logger)
	services := server.NewServices(config, logger)
	e := server.NewEchoServer(services, config, logger, validator)

	go func() {
		if err := e.Start(":" + config.Port); err != nil {
			if err == http.ErrServerClosed {
				logger.Info().Msg("user_account: server shutting down")
			}
			logger.Fatal().Err(err).Send()
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-shutdown
	cancelCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := e.Shutdown(cancelCtx); err != nil {
		logger.Fatal().Err(err).Send()
	}
}
