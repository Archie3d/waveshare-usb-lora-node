package meshtastic

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/log"
	pb "github.com/meshtastic/go/generated"
	"github.com/nats-io/nats.go"
)

type TextApplicationIncomingMessage struct {
	ChannelId uint32  `json:"channel"`
	From      uint32  `json:"from"`
	Text      string  `json:"text"`
	Rssi      int32   `json:"rssi"`
	Snr       float32 `json:"snr"`
}

type TextApplicationOutgoingMessage struct {
	ChannelId uint32 `json:"channel"`
	To        uint32 `json:"to"`
	Text      string `json:"text"`
}

type TextApplication struct {
	config          *NodeConfiguration
	natsConn        *nats.Conn
	messageSink     ApplicationMessageSink
	outgoingSubject string
	incomingSubject string
}

func NewTextApplication(config *NodeConfiguration) *TextApplication {
	return &TextApplication{
		config:          config,
		natsConn:        nil,
		messageSink:     nil,
		outgoingSubject: config.NatsSubjectPrefix + ".app.text_message.outgoing",
		incomingSubject: config.NatsSubjectPrefix + ".app.text_message.incoming",
	}
}

func (app *TextApplication) GetPortNum() pb.PortNum {
	return pb.PortNum_TEXT_MESSAGE_APP
}

func (app *TextApplication) Start(natsConnection *nats.Conn, sink ApplicationMessageSink) error {
	app.natsConn = natsConnection
	app.messageSink = sink

	app.natsConn.Subscribe(app.outgoingSubject, func(msg *nats.Msg) {
		var textMessage TextApplicationOutgoingMessage

		err := json.Unmarshal(msg.Data, &textMessage)
		if err != nil {
			log.With("err", err).Errorf("failed to unmarshal text message")
			return
		}

		err = app.messageSink.SendApplicationMessage(
			textMessage.ChannelId,
			textMessage.To,
			app.GetPortNum(),
			[]byte(textMessage.Text),
		)

		if err != nil {
			log.With("err", err).Errorf("failed to send text message")
		}
	})

	log.Info("Started Text application")

	return nil
}

func (app *TextApplication) Stop() error {
	return nil
}

func (app *TextApplication) HandleIncomingPacket(meshPacket *pb.MeshPacket) error {
	decoded, ok := meshPacket.PayloadVariant.(*pb.MeshPacket_Decoded)

	if !ok {
		return fmt.Errorf("invalid message format")
	}

	if app.natsConn != nil {
		textMessage := TextApplicationIncomingMessage{
			ChannelId: meshPacket.Channel,
			From:      meshPacket.From,
			Text:      string(decoded.Decoded.Payload),
			Rssi:      meshPacket.RxRssi,
			Snr:       meshPacket.RxSnr,
		}

		jsonMessage, err := json.Marshal(textMessage)

		if err != nil {
			return err
		}

		err = app.natsConn.Publish(app.incomingSubject, jsonMessage)
		if err != nil {
			return err
		}
	}

	return nil
}
