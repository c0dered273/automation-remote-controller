package server

import (
	"context"
	"encoding/json"

	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/model"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"
)

// TGBot содержит объект для взаимодействия с telegram api и реализует модель обработчиков для команд, поступающих от tg api
type TGBot struct {
	ctx          context.Context
	botApi       *tgbotapi.BotAPI
	notification chan model.Notification
	handler      MessageHandler
	logger       zerolog.Logger
}

// notify отправляет сообщение в telegram api
func (b *TGBot) notify(n model.Notification) error {
	msg := tgbotapi.NewMessage(n.ChatID, n.Text)
	if _, err := b.botApi.Send(msg); err != nil {
		return err
	}
	return nil
}

// GetNotifyChan отдает канал для отправки уведомлений в telegram
func (b *TGBot) GetNotifyChan() chan<- model.Notification {
	return b.notification
}

// ServeAndNotify запускает циклический опрос обновлений от telegram api,
// а также опрашивает внутренний канал отправки уведомлений и перенаправляет сообщения в telegram
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

// NewTGBot настраивает и возвращает настроенного бота
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

// MessageHandler описывает обработчик команд, поступающий от telegram api
type MessageHandler interface {
	ServeBotMessage(update tgbotapi.Update, botApi *tgbotapi.BotAPI)
}

// DefaultMessageHandler стандартная реализация обработчика команд
// хранит функции обработчики в двух словарях - для команд message и callback
type DefaultMessageHandler struct {
	logger       zerolog.Logger
	messages     map[string]func(update tgbotapi.Update, botApi *tgbotapi.BotAPI)
	callbacks    map[string]func(update tgbotapi.Update, botApi *tgbotapi.BotAPI)
	unknownRoute func(update tgbotapi.Update, botApi *tgbotapi.BotAPI)
}

// ServeBotMessage ищет совпадение полученной команды и ранее зарегистрированного обработчика
// если не находит, запускает дефолтный обработчик для неизвестной команды
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

// Message регистрирует обработчик для команды типа message
func (h *DefaultMessageHandler) Message(text string, handler func(update tgbotapi.Update, botApi *tgbotapi.BotAPI)) {
	if len(text) > 0 {
		h.messages[text] = handler
	}
}

// Callback регистрирует обработчик для команды типа callback
func (h *DefaultMessageHandler) Callback(handlerName string, handler func(update tgbotapi.Update, botApi *tgbotapi.BotAPI)) {
	if len(handlerName) > 0 {
		h.callbacks[handlerName] = handler
	}
}

// Unknown регистрирует обработчик для неизвестной команды
func (h *DefaultMessageHandler) Unknown(handler func(update tgbotapi.Update, botApi *tgbotapi.BotAPI)) {
	h.unknownRoute = handler
}

// NewMessageHandler возвращает новый контейнер обработчиков команд
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
