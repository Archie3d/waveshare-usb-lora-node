package meshtastic

import (
	"encoding/json"
	"fmt"

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
}

func NewNodeInfoApplication(config *NodeConfiguration) *NodeInfoApplication {
	return &NodeInfoApplication{
		config:          config,
		natsConn:        nil,
		messageSink:     nil,
		outgoingSubject: config.NatsSubjectPrefix + ".app.node_info.outgoing",
		incomingSubject: config.NatsSubjectPrefix + ".app.node_info.incoming",
	}
}

func (app *NodeInfoApplication) GetPortNum() pb.PortNum {
	return pb.PortNum_NODEINFO_APP
}

func (app *NodeInfoApplication) Start(natsConnection *nats.Conn, sink ApplicationMessageSink) {
	app.natsConn = natsConnection
	app.messageSink = sink
}

func (app *NodeInfoApplication) Stop() {
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
