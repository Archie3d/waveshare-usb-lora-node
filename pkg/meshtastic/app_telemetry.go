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

type TelemetryApplication struct {
	config          *NodeConfiguration
	natsConn        *nats.Conn
	messageSink     ApplicationMessageSink
	outgoingSubject string
	incomingSubject string
	startTime       time.Time

	wg        sync.WaitGroup
	eventLoop event_loop.EventLoop
}

func NewTelementryApplication(config *NodeConfiguration) *TelemetryApplication {
	return &TelemetryApplication{
		config:          config,
		natsConn:        nil,
		messageSink:     nil,
		outgoingSubject: config.NatsSubjectPrefix + ".out.telemetry",
		incomingSubject: config.NatsSubjectPrefix + ".in.telemetry",
		eventLoop:       event_loop.NewEventLoop(),
	}
}

func (app *TelemetryApplication) GetPortNum() pb.PortNum {
	return pb.PortNum_TELEMETRY_APP
}

func (app *TelemetryApplication) Start(natsConnection *nats.Conn, sink ApplicationMessageSink) error {
	app.natsConn = natsConnection
	app.messageSink = sink

	app.startTime = time.Now()

	app.wg.Go(func() {
		app.eventLoop.Run()
	})

	log.Info("Started Telemetry application")

	if app.config.Telemetry.DeviceMetrics != nil {
		app.eventLoop.Post(func(el event_loop.EventLoop) {
			app.publishDeviceMetrics()
		}, time.Now().Add(time.Duration(rand.Uint32N(20)+10)*time.Second))

		log.With(
			"channel", app.config.Telemetry.DeviceMetrics.Channel,
			"period", app.config.Telemetry.DeviceMetrics.PublishPeriod.String(),
		).Info("Started Device Metrics telemetry")

	}

	return nil
}

func (app *TelemetryApplication) Stop() error {
	app.eventLoop.Quit()
	app.wg.Wait()

	return nil
}

func (app *TelemetryApplication) HandleIncomingPacket(meshPacket *pb.MeshPacket) error {
	if app.natsConn == nil {
		return nil
	}

	decoded, ok := meshPacket.PayloadVariant.(*pb.MeshPacket_Decoded)

	if !ok {
		return fmt.Errorf("invalid messageformat")
	}

	var telemetry pb.Telemetry
	err := proto.Unmarshal(decoded.Decoded.Payload, &telemetry)
	if err != nil {
		return err
	}

	var subject string = app.incomingSubject
	var jsonData []byte = nil

	switch payload := telemetry.Variant.(type) {
	case *pb.Telemetry_DeviceMetrics:
		subject += ".device_metrics"
		jsonData, err = json.Marshal(payload.DeviceMetrics)
	case *pb.Telemetry_EnvironmentMetrics:
		subject += ".environment"
		jsonData, err = json.Marshal(payload.EnvironmentMetrics)
	default:
		return fmt.Errorf("Unsupported telemetry message %T", payload)
	}

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

	jsonData, err = json.Marshal(data)
	if err != nil {
		return err
	}

	return app.natsConn.Publish(
		subject,
		jsonData,
	)
}

func (app *TelemetryApplication) publishDeviceMetrics() {
	var batteryLevel uint32 = 101
	var voltage float32 = 5.0
	var channelUtilization float32 = 0.0 // %, @todo
	var airUntilTx float32 = 0.0         // %, @todo
	uptimeSeconds := uint32(time.Since(app.startTime).Seconds())

	telemetry := pb.Telemetry{
		Variant: &pb.Telemetry_DeviceMetrics{
			DeviceMetrics: &pb.DeviceMetrics{
				BatteryLevel:       &batteryLevel,
				Voltage:            &voltage,
				ChannelUtilization: &channelUtilization,
				AirUtilTx:          &airUntilTx,
				UptimeSeconds:      &uptimeSeconds,
			},
		},
	}

	bytes, err := proto.Marshal(&telemetry)

	if err != nil {
		log.With("err", err).Warn("Failed to marshal node info data")
		return
	}

	app.messageSink.SendApplicationMessage(
		app.config.Telemetry.DeviceMetrics.Channel, // Channel
		types.NodeId(0xFFFFFFFF),                   // Broadcast
		app.GetPortNum(),
		bytes,
	)

	publishPeriod := time.Duration(app.config.Telemetry.DeviceMetrics.PublishPeriod)

	app.eventLoop.Post(func(el event_loop.EventLoop) {
		app.publishDeviceMetrics()
	}, time.Now().Add(publishPeriod))
}
