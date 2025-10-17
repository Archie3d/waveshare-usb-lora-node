package meshtastic

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/event_loop"
	pb "github.com/meshtastic/go/generated"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type NodeInfoApplication struct {
	config          *NodeConfiguration
	natsConn        *nats.Conn
	messageSink     ApplicationMessageSink
	outgoingSubject string
	incomingSubject string

	wg        sync.WaitGroup
	eventLoop event_loop.EventLoop
}

func NewNodeInfoApplication(config *NodeConfiguration) *NodeInfoApplication {
	return &NodeInfoApplication{
		config:          config,
		natsConn:        nil,
		messageSink:     nil,
		outgoingSubject: config.NatsSubjectPrefix + ".app.node_info.outgoing",
		incomingSubject: config.NatsSubjectPrefix + ".app.node_info.incoming",
		eventLoop:       event_loop.NewEventLoop(),
	}
}

func (app *NodeInfoApplication) GetPortNum() pb.PortNum {
	return pb.PortNum_NODEINFO_APP
}

func (app *NodeInfoApplication) Start(natsConnection *nats.Conn, sink ApplicationMessageSink) {
	app.natsConn = natsConnection
	app.messageSink = sink

	app.wg.Go(func() {
		app.eventLoop.Run()
	})

	app.eventLoop.Post(func(el event_loop.EventLoop) {
		app.publishNodeInfo()
	}, time.Now().Add(10*time.Second))
}

func (app *NodeInfoApplication) Stop() {
	app.eventLoop.Quit()
	app.wg.Wait()
}

func (app *NodeInfoApplication) HandleIncomingPacket(meshPacket *pb.MeshPacket) error {
	decoded, ok := meshPacket.PayloadVariant.(*pb.MeshPacket_Decoded)

	if !ok {
		return fmt.Errorf("invalid message format")
	}

	var user pb.User
	err := proto.Unmarshal(decoded.Decoded.Payload, &user)
	if err != nil {
		return err
	}

	jsonMessage, err := json.Marshal(&user)

	if err != nil {
		return err
	}

	err = app.natsConn.Publish(app.incomingSubject, jsonMessage)
	if err != nil {
		return err
	}

	return nil
}

func (app *NodeInfoApplication) publishNodeInfo() {

	user := pb.User{
		Id:         fmt.Sprintf("!%x", app.config.Id),
		LongName:   app.config.LongName,
		ShortName:  app.config.ShortName,
		Macaddr:    app.config.MacAddress.AsByteArray(),
		HwModel:    pb.HardwareModel(app.config.HwModel),
		IsLicensed: false, // If true, then LongName must beoperator's licence number
		PublicKey:  app.config.PublicKey,
		Role:       pb.Config_DeviceConfig_CLIENT,
	}

	bytes, err := proto.Marshal(&user)
	if err != nil {
		log.Printf("Failed to marshal %s\n", err.Error())
		return
	}

	app.messageSink.SendApplicationMessage(
		uint32(0),          // Channel
		uint32(0xFFFFFFFF), // Broadcast
		app.GetPortNum(),
		bytes,
	)

	app.eventLoop.Post(func(el event_loop.EventLoop) {
		app.publishNodeInfo()
	}, time.Now().Add(1*time.Minute))
}
