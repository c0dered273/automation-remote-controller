package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	plc4go "github.com/apache/plc4x/plc4go/pkg/api"
	"github.com/apache/plc4x/plc4go/pkg/api/drivers"
	"github.com/c0dered273/automation-remote-controller/internal/common/loggers"
	"github.com/c0dered273/automation-remote-controller/internal/common/model"
	"github.com/c0dered273/automation-remote-controller/internal/common/proto"
	"github.com/c0dered273/automation-remote-controller/internal/remote-control-client/client"
	"github.com/c0dered273/automation-remote-controller/internal/remote-control-client/plc"
	"github.com/google/uuid"
)

func main() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	ctx, cancel := context.WithCancel(context.Background())

	config := client.ReadConfig()
	logger := loggers.NewLogger(client.LogWriter, config.Logger, "rc-client")
	c, err := client.NewClients(ctx, config, logger)
	if err != nil {
		logger.Err(err).Msg("rc-client: failed to init grpc clients")
	}

	// Client
	stream, err := c.EventMultiServiceClient.EventStreaming(c.Ctx)
	if err != nil {
		return
	}

	notifyChan := make(chan model.NotifyEvent, 1)
	actionChan := make(chan model.ActionEvent)

	go func() {
		for {
			select {
			case <-ctx.Done():
				err := stream.CloseSend()
				if err != nil {
					logger.Fatal().Err(err)
					return
				}
			case n := <-notifyChan:
				payload, _ := json.Marshal(n)
				req := proto.Event{
					Id:      uuid.NewString(),
					Action:  proto.Action_NOTIFICATION,
					Payload: payload,
				}

				err := stream.Send(&req)
				if err != nil {
					logger.Fatal().Err(err)
				}
				logger.Info().Str("action", req.Action.String()).RawJSON("payload", payload).Msg("send event")
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				resp, err := stream.Recv()
				if err != nil {
					logger.Fatal().Err(err)
				}
				if resp == nil {
					logger.Error().Msg("rc-client: lost connection")
					return
				}

				a := model.ActionEvent{}
				err = json.Unmarshal(resp.Payload, &a)
				if err != nil {
					logger.Error().Err(err)
					continue
				}
				actionChan <- a
				logger.Info().Str("action", resp.Action.String()).RawJSON("payload", resp.Payload).Msg("incoming event")
			}
		}
	}()

	// plc
	driverManager := plc4go.NewPlcDriverManager()
	drivers.RegisterModbusTcpDriver(driverManager)

	conn, err := plc.NewConnPool(driverManager, "modbus-tcp://10.0.1.10?unit-identifier=1&request-timeout=5000")
	if err != nil {
		fmt.Print(err)
		return
	}
	conn.SetMaxOpenConns(4)
	conn.SetConnTimeout(3 * time.Second)

	go func() {
		var lastTrigger0 bool
		var lastTrigger1 bool
		for {
			resp, err := conn.ReadTagAddress(ctx, "read1", "holding-register:1:WORD")
			if err != nil {
				fmt.Printf("read1 error, %s \n", err)
				continue
			}
			value := resp.GetValue("read1").GetBoolArray()

			if value[0] && !lastTrigger0 {
				notifyChan <- model.NotifyEvent{
					TGName: "c0dered",
					Text:   "Channel I0.0 active",
				}
				lastTrigger0 = true
			}
			lastTrigger0 = value[0]

			if value[1] && !lastTrigger1 {
				notifyChan <- model.NotifyEvent{
					TGName: "c0dered",
					Text:   "Channel I0.1 active",
				}
				lastTrigger1 = true
			}
			lastTrigger1 = value[1]
			time.Sleep(100 * time.Millisecond)
		}
	}()

	go func() {
		for {
			a := <-actionChan
			var value uint8
			switch a.Action {
			case model.SwitchON:
				value = 1
			case model.SwitchOFF:
				value = 0
			}

			_, err := conn.WriteTagAddress(ctx, "write1", "holding-register:1:WORD", value)
			if err != nil {
				fmt.Printf("write1 error, %s \n", err)
				continue
			}
		}
	}()

	//

	<-shutdown
	_ = conn.Close()
	cancel()
	logger.Info().Msg("Client shutting down")
}
