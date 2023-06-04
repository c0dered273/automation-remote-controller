package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/configs"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/model"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/services"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/users"
	"github.com/c0dered273/automation-remote-controller/pkg/collections"
	"github.com/c0dered273/automation-remote-controller/pkg/interceptors"
	"github.com/c0dered273/automation-remote-controller/pkg/loggers"
	"github.com/c0dered273/automation-remote-controller/pkg/proto"
	"github.com/c0dered273/automation-remote-controller/pkg/validators"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	LogWriter      = os.Stdout
	configFileName = "tgbot_config"
	configFilePath = []string{
		".",
		"./configs/",
	}
)

// ReadConfig формирует и валидирует конфигурацию приложения
func ReadConfig() *configs.TGBotCfg {
	logger := loggers.NewDefaultLogger(LogWriter)
	validator := validators.NewValidatorWithTagFieldName("mapstructure", logger)
	config, err := configs.NewTGBotConfig(configFileName, configFilePath, logger, validator)
	if err != nil {
		logger.Fatal().Err(err).Msg("rc-tg-bot: config init failed")
	}

	return config
}

// newServerCredentials читает с диска сертификаты и отдает настроенные реквизиты для TLS
// сервер требует наличие у клиента валидного сертификата, подписанного одним общим корневым сертификатом
func newServerCredentials(config *configs.TGBotCfg, logger zerolog.Logger) (credentials.TransportCredentials, error) {
	caPem, err := os.ReadFile(config.CACert)
	if err != nil {
		logger.Fatal().Err(err).Msg("remote-control-tg-bot: filed to read CA certificate")
		return nil, err
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPem) {
		logger.Fatal().Msg("remote-control-tg-bot: error loading CA to cert pool")
		return nil, err
	}
	serverCert, err := tls.LoadX509KeyPair(config.ServerCert, config.ServerPkey)
	if err != nil {
		logger.Fatal().Err(err).Msg("remote-control-tg-bot: failed to read server certificate")
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS13,
	}
	return credentials.NewTLS(tlsConfig), nil
}

// newServerOptions устанавливает опции для gRPC сервера
func newServerOptions(logger zerolog.Logger, creds credentials.TransportCredentials) []grpc.ServerOption {
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(interceptors.InterceptorLogger(logger), interceptors.GetLoggerOpts()...),
			recovery.UnaryServerInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			logging.StreamServerInterceptor(interceptors.InterceptorLogger(logger), interceptors.GetLoggerOpts()...),
			recovery.StreamServerInterceptor(),
		),
	}

	opts = append(opts, grpc.Creds(creds))
	return opts
}

// NewGRPCServer создает и настраивает gRPC сервер
func NewGRPCServer(
	ctx context.Context,
	config *configs.TGBotCfg,
	logger zerolog.Logger,
	clients *collections.ConcurrentMap[string, *model.ClientEvents],
	notify chan<- model.Notification,
	userService users.UserService,
) (*grpc.Server, error) {
	creds, err := newServerCredentials(config, logger)
	if err != nil {
		return nil, err
	}
	serverOptions := newServerOptions(logger, creds)
	server := grpc.NewServer(serverOptions...)

	proto.RegisterEventMultiServiceServer(server, services.NewEventMultiService(ctx, logger, clients, notify, userService))

	return server, err
}
