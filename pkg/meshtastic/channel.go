package meshtastic

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/client"
	pb "github.com/meshtastic/go/generated"
	"google.golang.org/protobuf/proto"
)

type Channel struct {
	id            uint32
	name          string
	encryptionKey []byte
	hash          byte
}

func NewChannel(id uint32, name string, key []byte) *Channel {

	var digest []byte = append([]byte(name), key...)

	var hash byte

	for i, c := range digest {
		if i == 0 {
			hash = c
		} else {
			hash = hash ^ c
		}
	}

	return &Channel{
		id:            id,
		name:          name,
		encryptionKey: key,
		hash:          hash,
	}
}

func (c *Channel) DecodePacket(packet *client.PacketReceived) (*pb.MeshPacket, error) {
	hash := packet.Data[13]
	if hash != c.hash {
		return nil, fmt.Errorf("channel hash mismatch")
	}

	toNode := binary.LittleEndian.Uint32(packet.Data[0:4])
	fromNode := binary.LittleEndian.Uint32(packet.Data[4:8])
	packetId := binary.LittleEndian.Uint32(packet.Data[8:12])

	nonce := make([]byte, 16)
	binary.LittleEndian.PutUint64(nonce[0:8], uint64(packetId))
	binary.LittleEndian.PutUint32(nonce[8:12], fromNode)

	encryptedPayload := packet.Data[16:]

	block, err := aes.NewCipher(c.encryptionKey)
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
		To:           toNode,
		Id:           packetId,
		PkiEncrypted: true,
		PublicKey:    c.encryptionKey,
		Channel:      c.id,
		RxRssi:       int32(packet.PacketRSSI_dBm),
		RxSnr:        float32(packet.PacketSNR_dB),
	}

	meshPacket.PayloadVariant = &pb.MeshPacket_Decoded{Decoded: data}

	return meshPacket, nil
}

func (c *Channel) EncodePacket(meshPacket *pb.MeshPacket) ([]byte, error) {
	packet := []byte{}

	binary.LittleEndian.AppendUint32(packet, meshPacket.To)
	binary.LittleEndian.AppendUint32(packet, meshPacket.From)
	binary.LittleEndian.AppendUint32(packet, meshPacket.Id)

	var flags byte = byte(meshPacket.HopLimit & 0x07)

	if meshPacket.WantAck {
		flags |= 0x08
	}

	if meshPacket.ViaMqtt {
		flags |= 0x10
	}

	flags |= byte(meshPacket.HopStart&0x07) << 5

	packet = append(packet, flags)
	packet = append(packet, c.hash)

	// @todo should we put the hop value in here?
	binary.LittleEndian.AppendUint16(packet, 0)

	block, err := aes.NewCipher(c.encryptionKey)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, 16)
	binary.LittleEndian.PutUint64(nonce[0:8], uint64(meshPacket.Id))
	binary.LittleEndian.PutUint32(nonce[8:12], meshPacket.From)

	decoded, ok := meshPacket.PayloadVariant.(*pb.MeshPacket_Decoded)
	if !ok {
		return nil, fmt.Errorf("unencrypted data expected")
	}

	payload, err := proto.Marshal(decoded.Decoded)

	if err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, nonce)
	encrypted := make([]byte, len(payload))
	stream.XORKeyStream(encrypted, payload)

	packet = append(packet, encrypted...)

	return packet, nil
}
