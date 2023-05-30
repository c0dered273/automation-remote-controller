package plc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	plc4go "github.com/apache/plc4x/plc4go/pkg/api"
	"github.com/c0dered273/automation-remote-controller/internal/remote-control-client/configs"
	"github.com/c0dered273/automation-remote-controller/pkg/model"
	"github.com/rs/zerolog"
)

func NewPlcConn(driverManager plc4go.PlcDriverManager, config *configs.RClientConfig) (*ConnPool, error) {
	conn, err := NewConnPool(driverManager, config.PLCUri)
	if err != nil {
		return nil, fmt.Errorf("failed to init plc connection pool: %w", err)
	}
	conn.SetMaxOpenConns(4)
	conn.SetConnTimeout(3 * time.Second)
	return conn, nil
}

type PollService struct {
	ctx         context.Context
	conn        *ConnPool
	sendChan    chan model.NotifyEvent
	receiveChan chan model.ActionEvent
	logger      zerolog.Logger
}

func (s PollService) Polling(config *configs.RClientConfig) {
	go func() {
		riseTrig := make(map[string]bool)
		for {
			for _, n := range config.Notifications {
				tag := strings.Split(n.TagAddress, "/")
				tagAddress := tag[0]
				bit, err := strconv.Atoi(tag[1])
				if err != nil {
					s.logger.Error().Err(err).Msgf("plc polling: failed to parse bit number: %s", tag[1])
					continue
				}

				resp, err := s.conn.ReadTagAddress(s.ctx, "read", tagAddress)
				if err != nil {
					s.logger.Error().Err(err).Msgf("plc polling: failed to read tag: %s", tagAddress)
					continue
				}
				value := resp.GetValue("read").GetBoolArray()

				// Rise trigger
				_, ok := riseTrig[n.TagAddress]
				if !ok {
					riseTrig[n.TagAddress] = false
				}
				if value[bit] && !riseTrig[n.TagAddress] {
					s.sendChan <- model.NotifyEvent{
						Text: n.Text[strconv.FormatBool(value[bit])],
					}
					riseTrig[n.TagAddress] = true
				}
				riseTrig[n.TagAddress] = value[bit]
				time.Sleep(100 * time.Millisecond)
			}

		}
	}()

	go func() {
		for {
			a := <-s.receiveChan
			for _, cfg := range config.Devices {
				if strings.EqualFold(a.DeviceID, cfg.DeviceID) {
					value, ok := cfg.Values[strings.ToLower(a.Action.String())]
					if !ok {
						s.logger.Error().Msgf("plc polling: action %s not found", a.Action.String())
					}

					_, err := s.conn.WriteTagAddress(s.ctx, "write", cfg.TagAddress, value)
					if err != nil {
						s.logger.Error().Err(err).Msgf("plc polling: failed to writing plc tag: %s, value %s", cfg.TagAddress, value)
						continue
					}
				}
			}
		}
	}()
}

func NewPLCPollService(
	ctx context.Context,
	conn *ConnPool,
	sendChan chan model.NotifyEvent,
	receiveChan chan model.ActionEvent,
	logger zerolog.Logger,
) *PollService {
	return &PollService{
		ctx:         ctx,
		conn:        conn,
		sendChan:    sendChan,
		receiveChan: receiveChan,
		logger:      logger,
	}
}
