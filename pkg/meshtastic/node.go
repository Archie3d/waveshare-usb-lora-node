package meshtastic

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/client"
	"github.com/Archie3d/waveshare-usb-lora-client/pkg/event_loop"
	pb "github.com/meshtastic/go/generated"
	"github.com/nats-io/nats.go"
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
	hwModel    uint32
	publicKey  []byte

	natsUrl  string
	natsConn *nats.Conn

	channels    []*Channel
	radioConfig RadioConfiguration

	serialPortName   string
	meshtasticClient *MeshtasticClient

	applications []Application

	eventLoop event_loop.EventLoop

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

		natsUrl:  config.NatsUrl,
		natsConn: nil,

		channels: []*Channel{
			NewChannel(0, defaultChannelName, defaultPublicKey),
		},

		radioConfig: config.Radio,

		serialPortName:   port,
		meshtasticClient: NewMeshtasticClient(),

		eventLoop: event_loop.NewEventLoop(),
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

func (n *Node) AddApplication(app Application) {
	n.applications = append(n.applications, app)
}

func (n *Node) Start() error {
	if err := n.meshtasticClient.Open(n.serialPortName, &n.radioConfig); err != nil {
		return err
	}

	// Run the nats client
	nc, err := nats.Connect(n.natsUrl)
	if err != nil {
		return err
	}

	n.natsConn = nc

	n.ctx, n.cancel = context.WithCancel(context.Background())

	n.wg.Go(func() {
	loop:
		for {
			select {
			case <-n.ctx.Done():
				break loop
			case packet := <-n.meshtasticClient.IncomingPackets:
				packetHandled := false
				isForThisNode := false

				for _, channel := range n.channels {
					meshPacket, err := channel.DecodePacket(packet)

					if err == nil && meshPacket != nil {
						isForThisNode = meshPacket.To == n.id
						n.handlePacket(meshPacket)
						packetHandled = true
						break
					}
				}

				if !packetHandled {
					n.handleUnknownPacket(packet)
				}

				if !isForThisNode {
					// This is not out packet - retransmit it
					n.retransmitPacket(packet)
				}
			}
		}
	})

	// Log errors from Meshtastic client
	n.wg.Go(func() {
	loop:
		for {
			select {
			case <-n.ctx.Done():
				break loop
			case err := <-n.meshtasticClient.Errors:
				log.Println(err.Error())
			}
		}
	})

	// Run the event loop
	n.wg.Go(n.eventLoop.Run)

	// Start the apps
	for _, app := range n.applications {
		app.Start(n.natsConn, n)
	}

	return nil
}

func (n *Node) Stop() error {
	if n.cancel == nil {
		return nil
	}

	// Stop the applications
	for _, app := range n.applications {
		app.Stop()
	}

	n.eventLoop.Quit()
	n.cancel()
	n.wg.Wait()

	return n.meshtasticClient.Close()
}

func (n *Node) GetChannel(channelId uint32) *Channel {
	for _, ch := range n.channels {
		if ch.id == channelId {
			return ch
		}
	}
	return nil
}

func (n *Node) SendText(channelId uint32, toNode uint32, text string) error {
	channel := n.GetChannel(channelId)

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

	// Retransmit
	n.eventLoop.Post(func(el event_loop.EventLoop) {
		n.meshtasticClient.OutgoingPackets <- data
	}, time.Now().Add(time.Second*3))

	n.eventLoop.Post(func(el event_loop.EventLoop) {
		n.meshtasticClient.OutgoingPackets <- data
	}, time.Now().Add(time.Second*7))

	return nil
}

// ApplicationMessageSink interface
func (n *Node) SendApplicationMessage(channelId uint32, destination uint32, portNum pb.PortNum, payload []byte) error {
	channel := n.GetChannel(channelId)

	if channel == nil {
		return fmt.Errorf("node does not have channel id %d", channelId)
	}

	meshPacket := pb.MeshPacket{
		From:     n.id,
		To:       destination,
		Channel:  channelId,
		Id:       rand.Uint32(), // @todo Have a better way to inject packet IDs
		WantAck:  false,
		ViaMqtt:  false,
		HopStart: 7,
		HopLimit: 7,
		PayloadVariant: &pb.MeshPacket_Decoded{
			Decoded: &pb.Data{
				Portnum: portNum,
				Payload: payload,
			},
		},
	}

	log.Printf("%v\n", meshPacket)

	if portNum == pb.PortNum_NODEINFO_APP {
		user := &pb.User{}
		err := proto.Unmarshal(payload, user)
		if err == nil {
			log.Printf("MY NODE INFO from %x: %v\n", meshPacket.From, user)
		}
	}

	data, err := channel.EncodePacket(&meshPacket)
	if err != nil {
		return err
	}

	n.meshtasticClient.OutgoingPackets <- data

	// Retransmit
	/*
		n.eventLoop.Post(func(el event_loop.EventLoop) {
			n.meshtasticClient.OutgoingPackets <- data
		}, time.Now().Add(time.Second*3))

		n.eventLoop.Post(func(el event_loop.EventLoop) {
			n.meshtasticClient.OutgoingPackets <- data
		}, time.Now().Add(time.Second*7))
	*/

	return nil
}

// Handle incoming packets received on the radio
func (n *Node) handlePacket(meshPacket *pb.MeshPacket) {
	decoded, ok := meshPacket.PayloadVariant.(*pb.MeshPacket_Decoded)
	if !ok {
		return
	}

	log.Printf("RSSI: %ddBm, SNR: %fdB\n", meshPacket.RxRssi, meshPacket.RxSnr)

	for _, app := range n.applications {
		if app.GetPortNum() == decoded.Decoded.Portnum {
			err := app.HandleIncomingPacket(meshPacket)
			if err != nil {
				log.Println(err)
			}
		}
	}

	switch decoded.Decoded.Portnum {
	case pb.PortNum_TEXT_MESSAGE_APP:
		log.Printf("TEXT MESSAGE from %x to %x: %s\n", meshPacket.From, meshPacket.To, decoded.Decoded.Payload)
		if n.natsConn != nil {
			n.natsConn.Publish("meshtastic.text", decoded.Decoded.Payload)
		}
	case pb.PortNum_NODEINFO_APP:
		user := &pb.User{}
		err := proto.Unmarshal(decoded.Decoded.Payload, user)
		if err == nil {
			log.Printf("NODE INFO from %x: %v\n", meshPacket.From, user)
		} else {
			log.Println("NODE INFO: unable to decode")
		}
	case pb.PortNum_TELEMETRY_APP:
		telemetry := &pb.Telemetry{}
		err := proto.Unmarshal(decoded.Decoded.Payload, telemetry)
		if err == nil {
			log.Printf("TELEMETRY %x: %v\n", meshPacket.From, telemetry)
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

func (n *Node) retransmitPacket(packet *client.PacketReceived) {
	flags := packet.Data[12]
	hopLimit := flags & 0x07

	if hopLimit == 0 {
		return
	}

	hopLimit -= 1

	flags = (flags & 0xF8) | hopLimit

	data := make([]byte, len(packet.Data))
	data[12] = flags

	log.Println("Retransmitting incoming packet")

	n.eventLoop.Post(func(el event_loop.EventLoop) {
		n.meshtasticClient.OutgoingPackets <- data
	}, time.Now().Add(time.Second))
}
