package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	plc4go "github.com/apache/plc4x/plc4go/pkg/api"
	"github.com/apache/plc4x/plc4go/pkg/api/drivers"
	"github.com/c0dered273/automation-remote-controller/internal/common/loggers"
	"github.com/c0dered273/automation-remote-controller/internal/remote-control-client/client"
	"github.com/c0dered273/automation-remote-controller/internal/remote-control-client/plc"
)

type NotifyEvent struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

func main() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	ctx, cancel := context.WithCancel(context.Background())

	config := client.ReadConfig()
	logger := loggers.NewLogger(client.LogWriter, config.Logger, "rc-client")
	//c, err := client.NewClients(ctx, config, logger)
	//if err != nil {
	//	logger.Err(err).Msg("rc-client: failed to init grpc clients")
	//}
	//
	//// Client
	//stream, err := c.EventMultiServiceClient.EventStreaming(c.Ctx)
	//if err != nil {
	//	return
	//}
	//
	//go func() {
	//	for {
	//		select {
	//		case <-ctx.Done():
	//			err := stream.CloseSend()
	//			if err != nil {
	//				logger.Fatal().Err(err)
	//				return
	//			}
	//		default:
	//			n := NotifyEvent{
	//				ID:   uuid.NewString(),
	//				Text: "FAKE_TEST_NOTIFICATION",
	//			}
	//			payload, _ := json.MarshalIndent(n, "", "  ")
	//
	//			req := proto.Event{
	//				Id:      uuid.NewString(),
	//				Action:  proto.Action_NOTIFICATION,
	//				Payload: payload,
	//			}
	//
	//			err := stream.Send(&req)
	//			if err != nil {
	//				logger.Fatal().Err(err)
	//			}
	//
	//			time.Sleep(5 * time.Second)
	//		}
	//	}
	//}()

	//go func() {
	//	for {
	//		select {
	//		case <-ctx.Done():
	//			return
	//		default:
	//			resp, err := stream.Recv()
	//			if err != nil {
	//				logger.Fatal().Err(err)
	//			}
	//
	//			if resp == nil {
	//				logger.Error().Msg("rc-client: lost connection")
	//				return
	//			}
	//
	//			fmt.Println("*** INCOMING EVENT")
	//			fmt.Println(resp.Id)
	//			fmt.Println(resp.Action)
	//			fmt.Println(string(resp.Payload))
	//		}
	//	}
	//}()

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

	for i := range [100]int{} {
		go func(n int) {
			for {
				resp, err := conn.ReadTagAddress(ctx, fmt.Sprint("READ-", n), "holding-register:1:WORD")
				if err != nil {
					fmt.Printf("read%d error, %s \n", n, err)
					continue
				}
				value := resp.GetValue(fmt.Sprint("READ-", n))

				fmt.Printf("READ-%d value: %d \n", n, value.GetInt16())
				time.Sleep(100 * time.Millisecond)
			}
		}(i)
	}

	// once
	//resp, err := conn.ReadTagAddress(ctx, "read1", "holding-register:1:WORD")
	//if err != nil {
	//	fmt.Printf("read1 error, %s", err)
	//	return
	//}
	//value := resp.GetValue("read1").GetInt16()
	//fmt.Printf("READ1 value: %d \n", value)

	go func() {
		for {
			resp, err := conn.ReadTagAddress(ctx, "read1", "holding-register:1:WORD")
			if err != nil {
				fmt.Printf("read1 error, %s \n", err)
				//if errors.Is(err, plc.ErrConnTimeout) {
				//	time.Sleep(1 * time.Second)
				//}
				continue
			}
			value := resp.GetValue("read1").GetInt16()

			fmt.Printf("READ1 value: %d \n", value)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	go func() {
		for {
			resp, err := conn.ReadTagAddress(ctx, "read2", "holding-register:2:WORD")
			if err != nil {
				fmt.Printf("read2 error, %s \n", err)
				continue
			}
			value := resp.GetValue("read2").GetInt16()

			fmt.Printf("READ2 value: %d \n", value)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	currentValue := 1
	go func() {
		for {
			resp, err := conn.WriteTagAddress(ctx, "write1", "holding-register:1:WORD", currentValue)
			if err != nil {
				fmt.Printf("write1 error, %s \n", err)
				continue
			}

			fmt.Printf("WRITE1: %v \n", resp.GetResponseCode("write1"))

			if currentValue == 1 {
				currentValue = 0
			} else {
				currentValue = 1
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	//

	<-shutdown
	conn.Close()
	cancel()
	logger.Info().Msg("Client shutting down")
}
