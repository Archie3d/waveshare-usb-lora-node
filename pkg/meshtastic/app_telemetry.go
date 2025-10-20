package meshtastic

import (
	"encoding/json"
	"fmt"

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
	config      *NodeConfiguration
	natsConn    *nats.Conn
	messageSink ApplicationMessageSink
}

func NewTelementryApplication(config *NodeConfiguration) *TelemetryApplication {
	return &TelemetryApplication{
		config:      config,
		natsConn:    nil,
		messageSink: nil,
	}
}

func (app *TelemetryApplication) GetPortNum() pb.PortNum {
	return pb.PortNum_TELEMETRY_APP
}

func (app *TelemetryApplication) Start(natsConnection *nats.Conn, sink ApplicationMessageSink) error {
	app.natsConn = natsConnection
	app.messageSink = sink

	app.natsConn.Subscribe(
		app.config.NatsSubjectPrefix+".app.telemetry.device_metrics.outgoing",
		func(msg *nats.Msg,
		) {

		})

	log.Info("Started Telemetry application")

	return nil
}

func (app *TelemetryApplication) Stop() error {
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
			app.config.NatsSubjectPrefix+".app.telemetry.device_metrics.incoming",
			jsonMessage,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
