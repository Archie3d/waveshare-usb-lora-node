package meshtastic

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/client"
	pb "github.com/meshtastic/go/generated"
	"google.golang.org/protobuf/proto"
)

type Header struct {
	Dest      uint32
	From      uint32
	Id        uint32
	Flags     byte
	Hash      byte
	NextHop   byte
	RelayNode byte
}

var DefaultPublicKey = []byte{0xd4, 0xf1, 0xbb, 0x3a, 0x20, 0x29, 0x07, 0x59, 0xf0, 0xbc, 0xff, 0xab, 0xcf, 0x4e, 0x69, 0x01}

type MeshtasticClient struct {
	apiClient *client.ApiClient
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	rssi_dBm atomic.Int32
}

func NewMeshtasticClient() *MeshtasticClient {
	return &MeshtasticClient{
		apiClient: client.NewApiClient(),
	}
}

func (c *MeshtasticClient) Open(portName string) error {
	err := c.apiClient.Open(portName)
	if err != nil {
		return err
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())

	c.initRadio()

	c.wg.Go(func() {
	loop:
		for {
			select {
			case <-c.ctx.Done():
				break loop
			case radioMessage := <-c.apiClient.Recv:
				c.handleRadioMessage(radioMessage)
			}
		}

		c.deinitRadio()
	})

	return nil
}

func (c *MeshtasticClient) Close() error {
	if c.cancel == nil {
		return nil
	}
	c.cancel()

	c.wg.Wait()

	return c.apiClient.Close()
}

func (c *MeshtasticClient) initRadio() {
	// Switch back to RX once the message has been transmitted
	_ = c.apiClient.SendRequest(&client.RxTxFallbackMode{FallbackMode: client.FALLBACK_STANDBY_XOSC_RX}, time.Second)

	// Start receiving
	c.switchToRx()
}

func (c *MeshtasticClient) deinitRadio() {
	_ = c.apiClient.SendRequest(&client.Standby{StandbyMode: client.STANDBY_XOSC}, time.Second)
}

func (c *MeshtasticClient) switchToRx() {
	_ = c.apiClient.SendRequest(&client.SwitchToRx{}, time.Second)
}

func (c *MeshtasticClient) handleRadioMessage(msg client.ApiMessage) {
	if packet, ok := msg.(*client.PacketReceived); ok {
		meshPacket, err := decodePacket(packet.Data, DefaultPublicKey)
		if err == nil {
			meshPacket.RxRssi = int32(packet.PacketRSSI_dBm)
			meshPacket.RxSnr = float32(packet.PacketSNR_dB)
			c.handleMeshPacket(meshPacket)
		}

		// Switch back to RX mode
		c.switchToRx()
	}

	if rssi, ok := msg.(*client.ContinuoisRSSI); ok {
		// Capture RSSI
		c.rssi_dBm.Store(int32(rssi.RSSI_dBm))
	}
}

func (c *MeshtasticClient) handleMeshPacket(meshPacket *pb.MeshPacket) {
	log.Printf("Packet received: %v\n", meshPacket)

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
	}
}

func decodePacket(packet []byte, key []byte) (*pb.MeshPacket, error) {
	fromNode := binary.LittleEndian.Uint32(packet[4:8])
	packetId := binary.LittleEndian.Uint32(packet[8:12])

	nonce := make([]byte, 16)
	binary.LittleEndian.PutUint64(nonce[0:8], uint64(packetId))
	binary.LittleEndian.PutUint32(nonce[8:12], fromNode)

	encryptedPayload := packet[16:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, nonce)
	decrypted := make([]byte, len(encryptedPayload))
	stream.XORKeyStream(decrypted, encryptedPayload)

	data := &pb.Data{}
	err = proto.Unmarshal(decrypted, data)
	if err != nil {
		return nil, err
	}

	meshPacket := &pb.MeshPacket{
		From:         fromNode,
		To:           binary.LittleEndian.Uint32(packet[0:4]),
		Id:           packetId,
		PkiEncrypted: true,
	}
	meshPacket.PayloadVariant = &pb.MeshPacket_Decoded{Decoded: data}

	return meshPacket, nil
}
