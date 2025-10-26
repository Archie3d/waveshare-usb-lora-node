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

type DeviceMetricsIncomingMessage struct {
	ChannelId          uint32       `json:"channel"`
	From               types.NodeId `json:"from"`
	Rssi               int32        `json:"rssi"`
	Snr                float32      `json:"snr"`
	BatteryLevel       uint32       `json:"battery_level"`
	Voltage            float32      `json:"voltage"`
	ChannelUtilization float32      `json:"channel_utilization"`
	AirUntilTx         float32      `json:"air_until_tx"`
	Uptime             uint32       `json:"uptime"`
}

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

	deviceMetrics, ok := telemetry.Variant.(*pb.Telemetry_DeviceMetrics)

	if ok {
		deviceMetricsMessage := DeviceMetricsIncomingMessage{
			ChannelId:          meshPacket.Channel,
			From:               types.NodeId(meshPacket.From),
			Rssi:               meshPacket.RxRssi,
			Snr:                meshPacket.RxSnr,
			BatteryLevel:       *deviceMetrics.DeviceMetrics.BatteryLevel,
			Voltage:            *deviceMetrics.DeviceMetrics.Voltage,
			ChannelUtilization: *deviceMetrics.DeviceMetrics.ChannelUtilization,
			AirUntilTx:         *deviceMetrics.DeviceMetrics.AirUtilTx,
			Uptime:             *deviceMetrics.DeviceMetrics.UptimeSeconds,
		}

		jsonMessage, err := json.Marshal(deviceMetricsMessage)
		if err != nil {
			return err
		}

		err = app.natsConn.Publish(
			app.incomingSubject+".device_metrics",
			jsonMessage,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (app *TelemetryApplication) publishDeviceMetrics() {

	var batteryLevel uint32 = 101
	var voltage float32 = 5.0
	var channelUtilization float32 = 100.0
	var airUntilTx float32 = 10.0
	uptimeSeconds := uint32(time.Since(app.startTime).Seconds())

	deviceMetrics := pb.DeviceMetrics{
		BatteryLevel:       &batteryLevel,
		Voltage:            &voltage,
		ChannelUtilization: &channelUtilization,
		AirUtilTx:          &airUntilTx,
		UptimeSeconds:      &uptimeSeconds,
	}

	bytes, err := proto.Marshal(&deviceMetrics)

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
