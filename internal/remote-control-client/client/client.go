package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"time"

	"github.com/c0dered273/automation-remote-controller/internal/common/interceptors"
	"github.com/c0dered273/automation-remote-controller/internal/common/loggers"
	"github.com/c0dered273/automation-remote-controller/internal/common/proto"
	"github.com/c0dered273/automation-remote-controller/internal/common/validators"
	"github.com/c0dered273/automation-remote-controller/internal/remote-control-client/configs"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

var (
	LogWriter      = os.Stdout
	configFileName = "remote-control-client"
	configFilePath = []string{
		".",
		"./configs/",
	}
)

type Clients struct {
	Ctx                     context.Context
	EventMultiServiceClient proto.EventMultiServiceClient
}

func ReadConfig() *configs.RClientConfig {
	logger := loggers.NewDefaultLogger(LogWriter)
	validator := validators.NewValidatorWithTagFieldName("mapstructure", logger)
	config, err := configs.NewRClientConfig(configFileName, configFilePath, logger, validator)
	if err != nil {
		logger.Fatal().Err(err).Msg("rc-client: config init failed")
	}

	return config
}

func newClientCredentials(config *configs.RClientConfig, logger zerolog.Logger) (credentials.TransportCredentials, error) {
	caPem, err := os.ReadFile(config.CACert)
	if err != nil {
		logger.Fatal().Err(err).Msg("rc-client: filed to read CA certificate")
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPem) {
		logger.Fatal().Msg("rc-client: error loading CA to cert pool")
		return nil, err
	}

	clientCert, err := tls.LoadX509KeyPair(config.ClientCert, config.ClientCert)
	if err != nil {
		logger.Fatal().Err(err).Msg("rc-client: failed to read server certificate")
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS13,
	}
	return credentials.NewTLS(tlsConfig), nil
}

func newConnection(creds credentials.TransportCredentials, config *configs.RClientConfig, logger zerolog.Logger) (*grpc.ClientConn, error) {
	//targetURL, err := url.Parse(config.ServerAddr)
	//if err != nil {
	//	return nil, err
	//}

	connectParams := grpc.ConnectParams{
		MinConnectTimeout: 15 * time.Second,
	}
	clientParams := keepalive.ClientParameters{
		Time:    30 * time.Second,
		Timeout: 60 * time.Second,
	}
	conn, err := grpc.Dial(config.ServerAddr,
		grpc.WithConnectParams(connectParams),
		grpc.WithKeepaliveParams(clientParams),
		grpc.WithTransportCredentials(creds),
		grpc.WithChainUnaryInterceptor(
			logging.UnaryClientInterceptor(interceptors.InterceptorLogger(logger), interceptors.GetLoggerOpts()...),
		),
		grpc.WithChainStreamInterceptor(
			logging.StreamClientInterceptor(interceptors.InterceptorLogger(logger), interceptors.GetLoggerOpts()...),
		),
	)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func NewClients(ctx context.Context, config *configs.RClientConfig, logger zerolog.Logger) (Clients, error) {
	creds, err := newClientCredentials(config, logger)
	if err != nil {
		return Clients{}, err
	}
	conn, err := newConnection(creds, config, logger)
	if err != nil {
		return Clients{}, err
	}

	return Clients{
		Ctx:                     ctx,
		EventMultiServiceClient: proto.NewEventMultiServiceClient(conn),
	}, nil
}
