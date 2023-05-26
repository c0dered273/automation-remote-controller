package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/c0dered273/automation-remote-controller/internal/common/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EventMultiService struct {
	proto.UnimplementedEventMultiServiceServer
}

type ToggleEvent struct {
	ID       string `json:"id"`
	DeviceID string `json:"device_id"`
}

func (s EventMultiService) EventStreaming(srv proto.EventMultiService_EventStreamingServer) error {
	for {
		req, err := srv.Recv()
		if err != nil {
			return status.Errorf(codes.Internal, "Internal error")
		}

		fmt.Println("*** INCOMING EVENT ***")
		fmt.Println(req.Id)
		fmt.Println(req.Action)
		fmt.Println(string(req.Payload))

		time.Sleep(1 * time.Second)

		t := ToggleEvent{
			ID:       uuid.NewString(),
			DeviceID: "FAKE_DEVICE_TOGGLE",
		}
		payload, _ := json.MarshalIndent(t, "", "  ")

		event := proto.Event{
			Id:      uuid.NewString(),
			Action:  proto.Action_TOGGLE,
			Payload: payload,
		}
		srv.Send(&event)
	}
}

func NewEventMultiService() EventMultiService {
	return EventMultiService{}
}
