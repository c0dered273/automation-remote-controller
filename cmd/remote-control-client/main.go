package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	plc4go "github.com/apache/plc4x/plc4go/pkg/api"
	"github.com/apache/plc4x/plc4go/pkg/api/drivers"
	"github.com/c0dered273/automation-remote-controller/internal/remote-control-client/client"
	"github.com/c0dered273/automation-remote-controller/internal/remote-control-client/plc"
	"github.com/c0dered273/automation-remote-controller/pkg/loggers"
	"github.com/c0dered273/automation-remote-controller/pkg/model"
)

//	@Title			remote-control-client
//	@Description	Брокер сообщений, конвертирует изменения тегов ПЛК в сообщения внутри gRPC стрима
//	@Version		0.0.1

func main() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	ctx, cancel := context.WithCancel(context.Background())

	config := client.ReadConfig()
	logger := loggers.NewLogger(client.LogWriter, config.Logger, "rc-client")
	sendChan := make(chan model.NotifyEvent, 1)
	receiveChan := make(chan model.ActionEvent)

	serverPoll := client.NewPollService(ctx, sendChan, receiveChan, config, logger)
	serverPoll.Poll()

	driverManager := plc4go.NewPlcDriverManager()
	drivers.RegisterModbusTcpDriver(driverManager)
	conn, err := plc.NewPlcConn(driverManager, config)
	if err != nil {
		logger.Fatal().Err(err)
	}
	plcPoll := plc.NewPLCPollService(ctx, conn, sendChan, receiveChan, logger)
	plcPoll.Polling(config)

	<-shutdown
	_ = conn.Close()
	cancel()
	logger.Info().Msg("Client shutting down")
}
