package main

import (
	"github.com/c0dered273/automation-remote-controller/internal/configs"
	"github.com/c0dered273/automation-remote-controller/internal/loggers"
	"github.com/c0dered273/automation-remote-controller/internal/validators"
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

	logger.Info().Msg("remote-control-tg-bot: started")
}
