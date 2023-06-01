package loggers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/c0dered273/automation-remote-controller/internal/common/configs"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

func init() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// NewDefaultLogger создает логгер с дефолтными настройками
// нужен для на этапе инициализации сервиса, пока рабочие настройки еще не получены
func NewDefaultLogger(w io.Writer) zerolog.Logger {
	prettyOutput := zerolog.ConsoleWriter{Out: w, TimeFormat: time.RFC3339}
	return zerolog.New(prettyOutput).With().Timestamp().Logger()
}

// NewLogger конструктор основного логгера
func NewLogger(w io.Writer, loggerConfig configs.Logger, serviceName string) zerolog.Logger {
	var logger zerolog.Logger

	if loggerConfig.Format == "pretty" {
		logger = NewDefaultLogger(w).With().Logger()
	} else {
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	if loggerConfig.Caller {
		logger = logger.With().Caller().Logger()
	}

	lvl, err := zerolog.ParseLevel(strings.ToLower(loggerConfig.Level))
	if err != nil {
		logger.Fatal().Msg("logger: failed to parse logging level")
	}
	logger = logger.Level(lvl)

	return logger.With().Str("appID", serviceName).Logger()
}

// EchoLogger обертка для адаптации zerolog к echo
type EchoLogger struct {
	output io.Writer
	logger zerolog.Logger
	prefix string
}

func (l *EchoLogger) Output() io.Writer {
	return l.output
}

func (l *EchoLogger) SetOutput(w io.Writer) {
	l.logger = l.logger.Output(w)
	l.output = w
}

func (l *EchoLogger) Prefix() string {
	return l.prefix
}

func (l *EchoLogger) SetPrefix(p string) {
	l.logger = l.logger.With().Str("prefix", p).Logger()
	l.prefix = p
}

func (l *EchoLogger) Level() log.Lvl {
	lvl := log.Lvl(l.logger.GetLevel())
	return lvl + 1
}

func (l *EchoLogger) SetLevel(v log.Lvl) {
	lvl := zerolog.Level(v) - 1
	l.logger = l.logger.Level(lvl)
}

func (l *EchoLogger) SetHeader(h string) {
}

func (l *EchoLogger) Print(i ...interface{}) {
	l.logger.Log().Msg(fmt.Sprint(i...))
}

func (l *EchoLogger) Printf(format string, args ...interface{}) {
	l.logger.Log().Msgf(format, args...)
}

func (l *EchoLogger) Printj(j log.JSON) {
	l.logger.Log().Msg(toString(j))
}

func (l *EchoLogger) Debug(i ...interface{}) {
	l.logger.Debug().Msg(fmt.Sprint(i...))
}

func (l *EchoLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debug().Msgf(format, args...)
}

func (l *EchoLogger) Debugj(j log.JSON) {
	l.logger.Debug().Msg(toString(j))
}

func (l *EchoLogger) Info(i ...interface{}) {
	l.logger.Info().Msg(fmt.Sprint(i...))
}

func (l *EchoLogger) Infof(format string, args ...interface{}) {
	l.logger.Info().Msgf(format, args...)
}

func (l *EchoLogger) Infoj(j log.JSON) {
	l.logger.Info().Msg(toString(j))
}

func (l *EchoLogger) Warn(i ...interface{}) {
	l.logger.Warn().Msg(fmt.Sprint(i...))
}

func (l *EchoLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warn().Msgf(format, args...)
}

func (l *EchoLogger) Warnj(j log.JSON) {
	l.logger.Warn().Msg(toString(j))
}

func (l *EchoLogger) Error(i ...interface{}) {
	l.logger.Error().Msg(fmt.Sprint(i...))
}

func (l *EchoLogger) Errorf(format string, args ...interface{}) {
	l.logger.Error().Msgf(format, args...)
}

func (l *EchoLogger) Errorj(j log.JSON) {
	l.logger.Error().Msg(toString(j))
}

func (l *EchoLogger) Fatal(i ...interface{}) {
	l.logger.Fatal().Msg(fmt.Sprint(i...))
}

func (l *EchoLogger) Fatalj(j log.JSON) {
	l.logger.Fatal().Msg(toString(j))
}

func (l *EchoLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatal().Msgf(format, args...)
}

func (l *EchoLogger) Panic(i ...interface{}) {
	l.logger.Panic().Msg(fmt.Sprint(i...))
}

func (l *EchoLogger) Panicj(j log.JSON) {
	l.logger.Panic().Msg(toString(j))
}

func (l *EchoLogger) Panicf(format string, args ...interface{}) {
	l.logger.Panic().Msgf(format, args...)
}

func toString(j log.JSON) string {
	bytes, err := json.MarshalIndent(j, "", "  ")
	message := string(bytes)
	if err != nil {
		message = fmt.Sprintf("logger: json marshalling error: %s", j)
	}
	return message
}

// NewEchoLogger конструктор для адаптера zerolog
func NewEchoLogger(w io.Writer, prefix string, logger zerolog.Logger) *EchoLogger {
	l := &EchoLogger{
		output: w,
		logger: logger,
	}
	l.SetPrefix(prefix)

	return l
}

// RequestLoggerMiddleware настраивает middleware для логгирования запросов к серверу echo
func RequestLoggerMiddleware(logger zerolog.Logger) func(next echo.HandlerFunc) echo.HandlerFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogHost:    true,
		LogURI:     true,
		LogStatus:  true,
		LogLatency: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info().
				Str("Host", v.Host).
				Str("URI", v.URI).
				Int("status", v.Status).
				Str("method", c.Request().Method).
				//Str("headers", fmt.Sprint(c.Request().Header)).
				Str("query_params", fmt.Sprint(c.Request().URL.Query())).
				Str("latency", fmt.Sprintf("%d ms", v.Latency.Milliseconds())).
				Msg("request")
			return nil
		},
	})
}
