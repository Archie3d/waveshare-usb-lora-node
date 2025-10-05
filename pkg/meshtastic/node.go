package meshtastic

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand/v2"
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

func NewNode(port string, config *NodeConfiguration) *Node {
	node := &Node{
		id:         config.Id,
		shortName:  config.ShortName,
		longName:   config.LongName,
		macAddress: config.MacAddress.AsByteArray(),
		hwModel:    config.HwModel,
		publicKey:  config.PublicKey,

		channels: []*Channel{
			NewChannel(0, defaultChannelName, defaultPublicKey),
		},

		serialPortName:   port,
		meshtasticClient: NewMeshtasticClient(),
	}

	for _, ch := range config.Channels {
		key := ch.EncryptionKey
		if len(key) == 1 && key[0] == 0x01 {
			key = defaultPublicKey
		}

		node.channels = append(node.channels, NewChannel(
			ch.Id,
			ch.Name,
			key,
		))
	}

	return node
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
			case packet := <-n.meshtasticClient.IncomingPackets:
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

func (n *Node) SendText(channelId uint32, toNode uint32, text string) error {
	var channel *Channel = nil

	for _, ch := range n.channels {
		if ch.id == channelId {
			channel = ch
			break
		}
	}

	if channel == nil {
		return fmt.Errorf("node does not have channel id %d", channelId)
	}

	meshPacket := pb.MeshPacket{
		From:     n.id,
		To:       toNode,
		Channel:  channelId,
		Id:       rand.Uint32(), // @todo Have a better way to inject packet IDs
		WantAck:  false,
		ViaMqtt:  false,
		HopStart: 7,
		HopLimit: 7,
		PayloadVariant: &pb.MeshPacket_Decoded{
			Decoded: &pb.Data{
				Portnum: pb.PortNum_TEXT_MESSAGE_APP,
				Payload: []byte(text),
			},
		},
	}

	data, err := channel.EncodePacket(&meshPacket)
	if err != nil {
		return err
	}

	n.meshtasticClient.OutgoingPackets <- data

	return nil
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
	log.Printf("handleUnknownPacket %v\n", packet)
	log.Println(hex.EncodeToString(packet.Data))
}
