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

type NodeInfoApplicationIncomingMessage struct {
	ChannelId  uint32       `json:"channel"`
	From       types.NodeId `json:"from"`
	Id         string       `json:"id"`
	LongName   string       `json:"long_name"`
	ShortName  string       `json:"short_name"`
	MacAddress string       `json:"mac_addess"`
	HwModel    uint32       `json:"hw_model"`
	PublicKey  string       `json:"public_key"`
	Rssi       int32        `json:"rssi"`
	Snr        float32      `json:"snr"`
}

type NodeInfoApplication struct {
	config          *NodeConfiguration
	natsConn        *nats.Conn
	messageSink     ApplicationMessageSink
	incomingSubject string
	publishPeriod   time.Duration

	wg        sync.WaitGroup
	eventLoop event_loop.EventLoop
}

func NewNodeInfoApplication(config *NodeConfiguration) *NodeInfoApplication {

	return &NodeInfoApplication{
		config:          config,
		natsConn:        nil,
		messageSink:     nil,
		incomingSubject: config.NatsSubjectPrefix + ".in.node_info",
		publishPeriod:   time.Duration(config.NodeInfo.PublishPeriod),
		eventLoop:       event_loop.NewEventLoop(),
	}
}

func (app *NodeInfoApplication) GetPortNum() pb.PortNum {
	return pb.PortNum_NODEINFO_APP
}

func (app *NodeInfoApplication) Start(natsConnection *nats.Conn, sink ApplicationMessageSink) error {
	app.natsConn = natsConnection
	app.messageSink = sink

	app.wg.Go(func() {
		app.eventLoop.Run()
	})

	app.eventLoop.Post(func(el event_loop.EventLoop) {
		app.publishNodeInfo()
	}, time.Now().Add(time.Duration(rand.Uint32N(20)+10)*time.Second))

	log.With(
		"channel", app.config.NodeInfo.Channel,
		"period", app.config.NodeInfo.PublishPeriod.String(),
	).Info("Started Node Info application")

	return nil
}

func (app *NodeInfoApplication) Stop() error {
	app.eventLoop.Quit()
	app.wg.Wait()

	return nil
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

	message := &NodeInfoApplicationIncomingMessage{
		ChannelId:  meshPacket.Channel,
		From:       types.NodeId(meshPacket.From),
		Id:         user.Id,
		LongName:   user.LongName,
		ShortName:  user.ShortName,
		MacAddress: types.MacAddress(user.Macaddr).String(),
		HwModel:    uint32(user.HwModel),
		PublicKey:  types.CryptoKey(user.PublicKey).String(),
		Rssi:       meshPacket.RxRssi,
		Snr:        meshPacket.RxSnr,
	}

	jsonMessage, err := json.Marshal(&message)

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
		Id:         fmt.Sprintf("!%s", app.config.Id),
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
		log.With("err", err).Warn("Failed to marshal node info data")
		return
	}

	app.messageSink.SendApplicationMessage(
		app.config.NodeInfo.Channel, // Channel
		types.NodeId(0xFFFFFFFF),    // Broadcast
		app.GetPortNum(),
		bytes,
	)

	app.eventLoop.Post(func(el event_loop.EventLoop) {
		app.publishNodeInfo()
	}, time.Now().Add(app.publishPeriod))
}
