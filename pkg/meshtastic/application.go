package meshtastic

import (
	"github.com/Archie3d/waveshare-usb-lora-client/pkg/types"
	pb "github.com/meshtastic/go/generated"
	"github.com/nats-io/nats.go"
)

type ApplicationMessageSink interface {
	SendApplicationMessage(channelId uint32, destination types.NodeId, portNum pb.PortNum, payload []byte) error
}

type Application interface {
	GetPortNum() pb.PortNum
	Start(natsConnection *nats.Conn, sink ApplicationMessageSink) error
	Stop() error
	HandleIncomingPacket(meshPacket *pb.MeshPacket) error
}
