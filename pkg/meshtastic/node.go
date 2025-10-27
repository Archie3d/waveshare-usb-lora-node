package meshtastic

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/client"
	"github.com/Archie3d/waveshare-usb-lora-client/pkg/event_loop"
	"github.com/Archie3d/waveshare-usb-lora-client/pkg/types"
	"github.com/charmbracelet/log"
	pb "github.com/meshtastic/go/generated"
	"github.com/nats-io/nats.go"
)

// AQ==
var defaultPublicKey = []byte{0xd4, 0xf1, 0xbb, 0x3a, 0x20, 0x29, 0x07, 0x59, 0xf0, 0xbc, 0xff, 0xab, 0xcf, 0x4e, 0x69, 0x01}

const defaultChannelName = "LongFast"

type Node struct {
	id         types.NodeId
	shortName  string
	longName   string
	macAddress []byte
	hwModel    uint32
	publicKey  []byte

	natsUrl           string
	natsConn          *nats.Conn
	natsSubjectPrefix string

	channels    []*Channel
	radioConfig RadioConfiguration

	serialPortName   string
	meshtasticClient *MeshtasticClient

	retransmitForward  bool
	retransmitPeriod   []types.Duration
	retransmitJitterMs uint32

	applications []Application

	eventLoop event_loop.EventLoop

	packetIdGenerator types.PacketIdGenerator

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

		natsUrl:           config.NatsUrl,
		natsConn:          nil,
		natsSubjectPrefix: config.NatsSubjectPrefix,

		channels: []*Channel{},

		radioConfig: config.Radio,

		serialPortName:   port,
		meshtasticClient: NewMeshtasticClient(),

		eventLoop: event_loop.NewEventLoop(),

		packetIdGenerator: *types.NewPacketIdGenerator(16),
	}

	if config.Retransmit != (*RetransmitConfiguration)(nil) {
		node.retransmitForward = config.Retransmit.Forward
		node.retransmitPeriod = config.Retransmit.Period
		node.retransmitJitterMs = uint32(time.Duration(config.Retransmit.Jitter) / time.Millisecond)
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

				log.With("packet", hex.EncodeToString(packet.Data)).Debug("Incoming")

				for _, channel := range n.channels {
					meshPacket, err := channel.DecodePacket(packet)

					if err == nil && meshPacket != nil {
						isForThisNode = types.NodeId(meshPacket.To) == n.id
						n.handlePacket(meshPacket)
						packetHandled = true
						break
					}
				}

				if !packetHandled {
					n.handleUnknownPacket(packet)
				}

				if !isForThisNode && n.retransmitForward {
					// This is not out packet - retransmit it
					n.eventLoop.Post(func(el event_loop.EventLoop) {
						n.retransmitPacket(packet)
					}, time.Now().Add(time.Duration(1000+rand.Uint32N(n.retransmitJitterMs))*time.Millisecond))
				}
			}
		}
	})

	// RSSI
	n.wg.Go(func() {
	loop:
		for {
			select {
			case <-n.ctx.Done():
				break loop
			case rssi := <-n.meshtasticClient.Rssi:
				if n.natsConn != nil {
					n.natsConn.Publish(
						fmt.Sprintf("%s.rssi", n.natsSubjectPrefix),
						[]byte(fmt.Sprintf("{\"timestamp\":%d, \"rssi\": %d}", time.Now().UnixMilli(), rssi)),
					)
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
				log.Error(err)
			case err := <-n.meshtasticClient.Warnings:
				log.Warn(err)
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

// ApplicationMessageSink interface
func (n *Node) SendApplicationMessage(channelId uint32, destination types.NodeId, portNum pb.PortNum, payload []byte) error {
	channel := n.GetChannel(channelId)

	if channel == nil {
		return fmt.Errorf("node does not have channel id %d", channelId)
	}

	log.With(
		"channel", channelId,
		"to", fmt.Sprintf("%08x", uint32(destination)),
		"from", fmt.Sprintf("%08x", uint32(n.id)),
	)

	meshPacket := pb.MeshPacket{
		From:     uint32(n.id),
		To:       uint32(destination),
		Channel:  channelId,
		Id:       n.packetIdGenerator.GetNext(),
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

	data, err := channel.EncodePacket(&meshPacket)
	if err != nil {
		return err
	}

	n.meshtasticClient.OutgoingPackets <- data

	log.With(
		"channel", channelId,
		"to", destination,
		"portNum", portNum,
	).Info("Outgoing packet")

	log.With("packet", hex.EncodeToString(data)).Debug("Outgoing")

	// Retransmit
	for _, period := range n.retransmitPeriod {
		n.eventLoop.Post(func(el event_loop.EventLoop) {
			n.meshtasticClient.OutgoingPackets <- data
		}, time.Now().Add(time.Duration(period)).Add(time.Duration(rand.Uint32N(n.retransmitJitterMs*uint32(time.Millisecond)))))
	}

	return nil
}

// Handle incoming packets received on the radio
func (n *Node) handlePacket(meshPacket *pb.MeshPacket) {
	decoded, ok := meshPacket.PayloadVariant.(*pb.MeshPacket_Decoded)
	if !ok {
		return
	}

	log.With(
		"RSSI", fmt.Sprintf("%ddBm", meshPacket.RxRssi),
		"SNR", fmt.Sprintf("%fdB", meshPacket.RxSnr),
		"From", fmt.Sprintf("%x", meshPacket.From),
		"To", fmt.Sprintf("%x", meshPacket.To),
		"Channel", meshPacket.Channel,
		"PortNum", decoded.Decoded.Portnum,
	).Debug("Received packet")

	for _, app := range n.applications {
		if app.GetPortNum() == decoded.Decoded.Portnum {
			err := app.HandleIncomingPacket(meshPacket)
			if err != nil {
				log.With("err", err).Error("Failed processing incoming packet")
			}
		}
	}

	/*
		switch decoded.Decoded.Portnum {
		case pb.PortNum_TEXT_MESSAGE_APP:
			log.Debugf("TEXT MESSAGE from %x to %x: %s\n", meshPacket.From, meshPacket.To, decoded.Decoded.Payload)
			if n.natsConn != nil {
				n.natsConn.Publish("meshtastic.text", decoded.Decoded.Payload)
			}
		case pb.PortNum_NODEINFO_APP:
			user := &pb.User{}
			err := proto.Unmarshal(decoded.Decoded.Payload, user)
			if err == nil {
				log.Debugf("NODE INFO from %x: %v\n", meshPacket.From, user)
			} else {
				log.Debug("NODE INFO: unable to decode")
			}
		case pb.PortNum_TELEMETRY_APP:
			telemetry := &pb.Telemetry{}
			err := proto.Unmarshal(decoded.Decoded.Payload, telemetry)
			if err == nil {
				log.Debugf("TELEMETRY %x: %v\n", meshPacket.From, telemetry)
			} else {
				log.Debug("TELEMETRY: unable to decode")
			}
		default:
			log.Printf("Unknown message: %v\n", decoded)
		}
	*/
}

func (n *Node) handleUnknownPacket(packet *client.PacketReceived) {
	log.With("packet", hex.EncodeToString(packet.Data)).Debug("Unhandled")
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
	copy(data, packet.Data)
	data[12] = flags

	log.Debug("Retransmitting incoming packet")

	n.eventLoop.Post(func(el event_loop.EventLoop) {
		n.meshtasticClient.OutgoingPackets <- data
	}, time.Now().Add(time.Second))
}
