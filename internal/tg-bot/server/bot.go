package server

import (
	"context"
	"encoding/json"

	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/model"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"
)

type TGBot struct {
	ctx          context.Context
	botApi       *tgbotapi.BotAPI
	notification chan model.Notification
	handler      MessageHandler
	logger       zerolog.Logger
}

func (b *TGBot) notify(n model.Notification) error {
	msg := tgbotapi.NewMessage(n.ChatID, n.Text)
	if _, err := b.botApi.Send(msg); err != nil {
		return err
	}
	return nil
}

func (b *TGBot) GetNotifyChan() chan<- model.Notification {
	return b.notification
}

func (b *TGBot) ServeAndNotify() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.botApi.GetUpdatesChan(u)
	go func() {
		for update := range updates {
			select {
			default:
			case <-b.ctx.Done():
				return
			}
			b.handler.ServeBotMessage(update, b.botApi)
		}
	}()
	go func() {
		for {
			select {
			case <-b.ctx.Done():
				return
			case n := <-b.notification:
				err := b.notify(n)
				if err != nil {
					b.logger.Error().Err(err).Send()
				}
				continue
			}
		}
	}()
}

func NewTGBot(ctx context.Context, token string, handler MessageHandler, logger zerolog.Logger) (*TGBot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	// bot.Debug = true

	logger.Info().Msgf("remote-control-tg-bot: authorized on account: %s", bot.Self.UserName)
	return &TGBot{
		ctx:          ctx,
		botApi:       bot,
		notification: make(chan model.Notification, 1),
		handler:      handler,
		logger:       logger,
	}, nil
}

type MessageHandler interface {
	ServeBotMessage(update tgbotapi.Update, botApi *tgbotapi.BotAPI)
}

type DefaultMessageHandler struct {
	logger       zerolog.Logger
	messages     map[string]func(update tgbotapi.Update, botApi *tgbotapi.BotAPI)
	callbacks    map[string]func(update tgbotapi.Update, botApi *tgbotapi.BotAPI)
	unknownRoute func(update tgbotapi.Update, botApi *tgbotapi.BotAPI)
}

func (h *DefaultMessageHandler) ServeBotMessage(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
	if update.Message != nil {
		handler, ok := h.messages[update.Message.Text]
		if ok {
			handler(update, botApi)
			return
		}
	} else if update.CallbackQuery != nil {
		handlerName := utils.ParseReqHandler(update.CallbackQuery.Data)
		handler, ok := h.callbacks[handlerName]
		if ok {
			handler(update, botApi)
			return
		}
	}
	h.unknownRoute(update, botApi)
}

func (h *DefaultMessageHandler) Message(text string, handler func(update tgbotapi.Update, botApi *tgbotapi.BotAPI)) {
	if len(text) > 0 {
		h.messages[text] = handler
	}
}

func (h *DefaultMessageHandler) Callback(handlerName string, handler func(update tgbotapi.Update, botApi *tgbotapi.BotAPI)) {
	if len(handlerName) > 0 {
		h.callbacks[handlerName] = handler
	}
}

func NewMessageHandler(logger zerolog.Logger) *DefaultMessageHandler {
	return &DefaultMessageHandler{
		logger:    logger,
		messages:  make(map[string]func(update tgbotapi.Update, botApi *tgbotapi.BotAPI)),
		callbacks: make(map[string]func(update tgbotapi.Update, botApi *tgbotapi.BotAPI)),
		unknownRoute: func(update tgbotapi.Update, botApi *tgbotapi.BotAPI) {
			updateJson, err := json.Marshal(update)
			if err != nil {
				updateJson = []byte("marshalling error")
			}
			logger.Error().RawJSON("update", updateJson).Msg("tg-bot: unknown route")
		},
	}
}
