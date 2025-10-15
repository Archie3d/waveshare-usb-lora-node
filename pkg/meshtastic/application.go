package meshtastic

import (
	pb "github.com/meshtastic/go/generated"
	"github.com/nats-io/nats.go"
)

type ApplicationMessageSink interface {
	SendApplicationMessage(channelId uint32, destination uint32, portNum pb.PortNum, payload []byte) error
}

type Application interface {
	GetPortNum() pb.PortNum
	Start(natsConnection *nats.Conn, sink ApplicationMessageSink)
	Stop()
	HandleIncomingPacket(meshPacket *pb.MeshPacket) error
}
