package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/c0dered273/automation-remote-controller/internal/remote-control-client/configs"
	"github.com/c0dered273/automation-remote-controller/pkg/interceptors"
	"github.com/c0dered273/automation-remote-controller/pkg/loggers"
	"github.com/c0dered273/automation-remote-controller/pkg/model"
	"github.com/c0dered273/automation-remote-controller/pkg/proto"
	"github.com/c0dered273/automation-remote-controller/pkg/validators"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
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

func newClientCredentials(config *configs.RClientConfig) (credentials.TransportCredentials, error) {
	caPem, err := os.ReadFile(config.CACert)
	if err != nil {
		return nil, fmt.Errorf("filed to read CA certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPem) {
		return nil, fmt.Errorf("error loading CA to cert pool: %w", err)
	}

	clientCert, err := tls.LoadX509KeyPair(config.ClientCert, config.ClientCert)
	if err != nil {
		return nil, fmt.Errorf("failed to read client certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS13,
	}
	return credentials.NewTLS(tlsConfig), nil
}

func newConnection(creds credentials.TransportCredentials, config *configs.RClientConfig, logger zerolog.Logger) (*grpc.ClientConn, error) {
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

func newClients(config *configs.RClientConfig, logger zerolog.Logger) (proto.EventMultiServiceClient, error) {
	creds, err := newClientCredentials(config)
	if err != nil {
		return nil, err
	}
	conn, err := newConnection(creds, config, logger)
	if err != nil {
		return nil, err
	}

	return proto.NewEventMultiServiceClient(conn), nil
}

func NewBidirectionalStream(ctx context.Context, config *configs.RClientConfig, logger zerolog.Logger) (proto.EventMultiService_EventStreamingClient, error) {
	c, err := newClients(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to init grpc client: %w", err)
	}

	md := metadata.New(map[string]string{
		"X-Username": config.TGUsername,
		"X-ClientID": config.CertID,
	})
	outCtx := metadata.NewOutgoingContext(ctx, md)
	return c.EventStreaming(outCtx)
}

type PollService struct {
	ctx         context.Context
	stream      proto.EventMultiService_EventStreamingClient
	sendChan    chan model.NotifyEvent
	receiveChan chan model.ActionEvent
	logger      zerolog.Logger
}

func (s *PollService) Poll() {
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				err := s.stream.CloseSend()
				if err != nil {
					s.logger.Fatal().Err(err)
					return
				}
			case n := <-s.sendChan:
				payload, _ := json.Marshal(n)
				req := proto.Event{
					Id:      uuid.NewString(),
					Action:  proto.Action_NOTIFICATION,
					Payload: payload,
				}

				err := s.stream.Send(&req)
				if err != nil {
					s.logger.Fatal().Err(err)
				}
				s.logger.Info().Str("action", req.Action.String()).RawJSON("payload", payload).Msg("send event")
			}
		}
	}()

	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				resp, err := s.stream.Recv()
				if err != nil {
					s.logger.Fatal().Err(err)
				}
				if resp == nil {
					s.logger.Error().Msg("rc-client: lost connection")
					return
				}

				a := model.ActionEvent{}
				err = json.Unmarshal(resp.Payload, &a)
				if err != nil {
					s.logger.Error().Err(err)
					continue
				}
				s.receiveChan <- a
				s.logger.Info().Str("action", resp.Action.String()).RawJSON("payload", resp.Payload).Msg("incoming event")
			}
		}
	}()
}

func NewPollService(
	ctx context.Context,
	stream proto.EventMultiService_EventStreamingClient,
	sendChan chan model.NotifyEvent,
	receiveChan chan model.ActionEvent,
	logger zerolog.Logger,
) *PollService {
	return &PollService{
		ctx:         ctx,
		stream:      stream,
		sendChan:    sendChan,
		receiveChan: receiveChan,
		logger:      logger,
	}
}
