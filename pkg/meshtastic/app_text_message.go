package meshtastic

import (
	"encoding/json"
	"fmt"

	pb "github.com/meshtastic/go/generated"
	"github.com/nats-io/nats.go"
)

type TextApplicationIncomingMessage struct {
	ChannelId uint32 `json:"channel"`
	From      uint32 `json:"from"`
	Text      string `json:"text"`
}

type TextApplicationOutgoingMessage struct {
	ChannelId uint32 `json:"channel"`
	To        uint32 `json:"to"`
	Text      string `json:"text"`
}

type TextApplication struct {
	natsConn    *nats.Conn
	messageSink ApplicationMessageSink
}

func NewTextApplication() *TextApplication {
	return &TextApplication{
		natsConn:    nil,
		messageSink: nil,
	}
}

func (app *TextApplication) GetPortNum() pb.PortNum {
	return pb.PortNum_TEXT_MESSAGE_APP
}

func (app *TextApplication) Start(natsConnection *nats.Conn, sink ApplicationMessageSink) {
	app.natsConn = natsConnection
	app.messageSink = sink

	app.natsConn.Subscribe("mesh.app.text_message.outgoing", func(msg *nats.Msg) {
		var textMessage TextApplicationOutgoingMessage

		if err := json.Unmarshal(msg.Data, &textMessage); err == nil {
			app.messageSink.SendApplicationMessage(
				textMessage.ChannelId,
				textMessage.To,
				app.GetPortNum(),
				[]byte(textMessage.Text),
			)
		}
	})
}

func (app *TextApplication) Stop() {
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
		}

		j, err := json.Marshal(textMessage)

		if err == nil {
			app.natsConn.Publish("mesh.app.text_message.incoming", j)
		}
	}

	return nil
}
