package server

import (
	"encoding/json"
	"net/url"

	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/configs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"
)

type TGBot struct {
	botApi  *tgbotapi.BotAPI
	handler MessageHandler
}

func (b *TGBot) Serve() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.botApi.GetUpdatesChan(u)
	for update := range updates {
		b.handler.ServeBotMessage(update, b.botApi)
	}
}

func (b *TGBot) Notification() func(chatID int64, text string) error {
	return func(chatID int64, text string) error {
		msg := tgbotapi.NewMessage(chatID, text)
		if _, err := b.botApi.Send(msg); err != nil {
			return err
		}
		return nil
	}
}

func NewTGBot(config *configs.TGBotCfg, logger zerolog.Logger, handler MessageHandler) (*TGBot, error) {
	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, err
	}

	bot.Debug = true

	logger.Info().Msgf("remote-control-tg-bot: authorized on account: %s", bot.Self.UserName)
	return &TGBot{
		botApi:  bot,
		handler: handler,
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
		handlerName := ParseReqHandler(update.CallbackQuery.Data)
		handler, ok := h.callbacks[handlerName]
		if ok {
			handler(update, botApi)
			return
		}
	}
	h.unknownRoute(update, botApi)
	return
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

func ParseReqHandler(reqURL string) string {
	reqUrl, err := url.Parse(reqURL)
	if err != nil {
		return ""
	}
	return reqUrl.Opaque
}

func ParseReqParams(reqURL string) map[string][]string {
	empty := make(map[string][]string)
	reqUrl, err := url.Parse(reqURL)
	if err != nil {
		return empty
	}
	params, err := url.ParseQuery(reqUrl.RawQuery)
	if err != nil {
		return empty
	}
	return params
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
