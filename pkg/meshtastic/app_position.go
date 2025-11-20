package meshtastic

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/event_loop"
	"github.com/Archie3d/waveshare-usb-lora-client/pkg/types"
	"github.com/charmbracelet/log"
	pb "github.com/meshtastic/go/generated"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type PositionApplication struct {
	config          *NodeConfiguration
	natsConn        *nats.Conn
	messageSink     ApplicationMessageSink
	outgoingSubject string
	incomingSubject string
	publishPeriod   time.Duration

	wg        sync.WaitGroup
	eventLoop event_loop.EventLoop
}

func NewPositionApplication(config *NodeConfiguration) *PositionApplication {
	return &PositionApplication{
		config:          config,
		natsConn:        nil,
		messageSink:     nil,
		outgoingSubject: config.NatsSubjectPrefix + ".out.position",
		incomingSubject: config.NatsSubjectPrefix + ".in.position",
		publishPeriod:   time.Duration(config.Position.PublishPeriod),
		eventLoop:       event_loop.NewEventLoop(),
	}
}

func (app *PositionApplication) GetPortNum() pb.PortNum {
	return pb.PortNum_POSITION_APP
}

func (app *PositionApplication) Start(natsConnection *nats.Conn, sink ApplicationMessageSink) error {
	app.natsConn = natsConnection
	app.messageSink = sink

	app.wg.Go(func() {
		app.eventLoop.Run()
	})

	app.eventLoop.Post(func(el event_loop.EventLoop) {
		app.publishNodePosition()
	}, time.Now().Add(time.Duration(rand.Uint32N(20)+10)*time.Second))

	log.With(
		"channel", app.config.Position.Channel,
		"period", app.config.Position.PublishPeriod.String(),
	).Info("Started Node Position application")

	return nil
}

func (app *PositionApplication) Stop() error {
	app.eventLoop.Quit()
	app.wg.Wait()

	return nil
}

func (app *PositionApplication) HandleIncomingPacket(meshPacket *pb.MeshPacket) error {
	if app.natsConn == nil {
		return nil
	}

	decoded, ok := meshPacket.PayloadVariant.(*pb.MeshPacket_Decoded)

	if !ok {
		return fmt.Errorf("invalid message format")
	}

	var position pb.Position
	err := proto.Unmarshal(decoded.Decoded.Payload, &position)
	if err != nil {
		return err
	}

	var jsonData []byte = nil

	jsonData, err = json.Marshal(&position)

	if err != nil {
		return err
	}

	var data map[string]interface{}
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return err
	}

	data["channel"] = meshPacket.Channel
	data["from"] = types.NodeId(meshPacket.From)
	data["rssi"] = meshPacket.RxRssi
	data["snr"] = meshPacket.RxSnr
	data["hops"] = meshPacket.HopStart - meshPacket.HopLimit

	jsonData, err = json.Marshal(data)
	if err != nil {
		return err
	}

	return app.natsConn.Publish(
		app.incomingSubject,
		jsonData,
	)
}

func (app *PositionApplication) publishNodePosition() {

	var latitude int32 = int32(app.config.Position.Latitude * 1e7)
	var longitude int32 = int32(app.config.Position.Longitude * 1e7)
	var altitude int32 = int32(app.config.Position.Altitude)

	position := pb.Position{
		LatitudeI:      &latitude,
		LongitudeI:     &longitude,
		Altitude:       &altitude,
		LocationSource: pb.Position_LOC_MANUAL,
		Time:           uint32(time.Now().Unix()),
	}

	bytes, err := proto.Marshal(&position)
	if err != nil {
		log.With("err", err).Warn("Failed to marshal node position data")
		return
	}

	app.messageSink.SendApplicationMessage(
		app.config.NodeInfo.Channel, // Channel
		types.NodeId(0xFFFFFFFF),    // Broadcast
		app.GetPortNum(),
		bytes,
	)

	app.eventLoop.Post(func(el event_loop.EventLoop) {
		app.publishNodePosition()
	}, time.Now().Add(app.publishPeriod))
}
