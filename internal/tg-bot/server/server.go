package server

import (
	"context"

	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/handlers"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/model"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/services"
	"github.com/c0dered273/automation-remote-controller/pkg/collections"
	"github.com/rs/zerolog"
)

func NewBotServer(
	ctx context.Context,
	token string,
	s services.Services,
	clientsMap *collections.ConcurrentMap[string, *model.ClientEvents],
	logger zerolog.Logger,
) *TGBot {
	// tg bot
	h := NewMessageHandler(logger)
	h.Message("/menu", handlers.MenuHandler(ctx, logger, s.UserService))
	h.Message("/start", handlers.StartNotificationsHandler(ctx, logger, s.UserService, clientsMap))
	h.Message("/stop", handlers.StopNotificationsHandler(ctx, logger, s.UserService, clientsMap))
	h.Callback("status", handlers.StatusHandler(ctx, logger, s.UserService))
	h.Callback("lightControl", handlers.LightControlHandler(ctx, logger, s.UserService))
	h.Callback("lampMenu", handlers.LampMenuHandler(ctx, logger, s.UserService))
	h.Callback("lampSwitch", handlers.LampSwitchHandler(ctx, logger, s.UserService, clientsMap))

	bot, err := NewTGBot(ctx, token, h, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("remote-control-tg-bot: bot init error")
	}

	return bot
}
