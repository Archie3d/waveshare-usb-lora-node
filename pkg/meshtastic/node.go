package meshtastic

import (
	"context"
	"log"
	"sync"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/client"
	pb "github.com/meshtastic/go/generated"
	"google.golang.org/protobuf/proto"
)

// AQ==
var defaultPublicKey = []byte{0xd4, 0xf1, 0xbb, 0x3a, 0x20, 0x29, 0x07, 0x59, 0xf0, 0xbc, 0xff, 0xab, 0xcf, 0x4e, 0x69, 0x01}

const defaultChannelName = "LongFast"

type Node struct {
	id         uint32
	shortName  string
	longName   string
	macAddress []byte
	hwModel    string
	publicKey  []byte

	channels []*Channel

	serialPortName   string
	meshtasticClient *MeshtasticClient

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewNode() *Node {
	// @todo This should be loaded from a configuration file
	return &Node{
		id:         0x12345678,
		shortName:  "WSH",
		longName:   "Waveshare",
		macAddress: []byte{0xAA, 0xBB, 0x12, 0x34, 0x56, 0x78}, // Last 4 bytes are the ID
		hwModel:    "Waveshare USB LoRa",
		publicKey:  []byte{0x00},

		channels: []*Channel{
			NewChannel(0, defaultChannelName, defaultPublicKey),
		},

		serialPortName:   "COM4",
		meshtasticClient: NewMeshtasticClient(),
	}
}

func (n *Node) Start() error {
	if err := n.meshtasticClient.Open(n.serialPortName); err != nil {
		return err
	}

	n.ctx, n.cancel = context.WithCancel(context.Background())

	n.wg.Go(func() {
	loop:
		for {
			select {
			case <-n.ctx.Done():
				break loop
			case packet := <-n.meshtasticClient.Packets:
				packetHandled := false
				for _, channel := range n.channels {
					meshPacket, err := channel.DecodePacket(packet)

					if err == nil && meshPacket != nil {
						n.handlePacket(meshPacket)
						packetHandled = true
						break
					}
				}

				if !packetHandled {
					n.handleUnknownPacket(packet)
				}
			}
		}
	})

	return nil
}

func (n *Node) Stop() error {
	if n.cancel == nil {
		return nil
	}

	n.cancel()
	n.wg.Wait()

	return n.meshtasticClient.Close()
}

func (n *Node) handlePacket(meshPacket *pb.MeshPacket) {
	decoded, ok := meshPacket.PayloadVariant.(*pb.MeshPacket_Decoded)
	if !ok {
		return
	}

	switch decoded.Decoded.Portnum {
	case pb.PortNum_TEXT_MESSAGE_APP:
		log.Printf("TEXT MESSAGE: %s\n", decoded.Decoded.Payload)
	case pb.PortNum_NODEINFO_APP:
		user := &pb.User{}
		err := proto.Unmarshal(decoded.Decoded.Payload, user)
		if err == nil {
			log.Printf("NODE INFO: %v\n", user)
		} else {
			log.Println("NODE INFO: unable to decode")
		}
	case pb.PortNum_TELEMETRY_APP:
		telemetry := &pb.Telemetry{}
		err := proto.Unmarshal(decoded.Decoded.Payload, telemetry)
		if err == nil {
			log.Printf("TELEMETRY: %v\n", telemetry)
		} else {
			log.Println("TELEMETRY: unable to decode")
		}
	default:
		log.Printf("Unknown message: %v\n", decoded)
	}
}

func (n *Node) handleUnknownPacket(packet *client.PacketReceived) {
	log.Printf("@todo handleUnknownPacket %v\n", packet)
}
