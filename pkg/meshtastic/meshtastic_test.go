package meshtastic

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"testing"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/client"
	pb "github.com/meshtastic/go/generated"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestDecode(t *testing.T) {
	defaultPublicKey := []byte{0xd4, 0xf1, 0xbb, 0x3a, 0x20, 0x29, 0x07, 0x59, 0xf0, 0xbc, 0xff, 0xab, 0xcf, 0x4e, 0x69, 0x01}

	// "Hello" message
	packet := []byte{
		0xFF, 0xFF, 0xFF, 0xFF, // Dest
		0xC0, 0x74, 0x3C, 0x43, // From (nodeNumber, little endian)
		0xBC, 0xA0, 0x4A, 0x11, // Id
		0xE7, // Flags
		0x08, // Hash - xor of the key (16 bytes) and the channel name, default channel name is ""
		0x00,
		0x00, // Align

		// Meshtastic payload
		33, 147, 1, 47, 199, 0, 80, 234, 167, 167, 95,
	}
	/*
		anotherPacket := []byte{
			255, 255, 255, 255, 192, 116, 60, 67,
			75, 9, 229, 73, 231, 8, 0, 0,
			77, 140, 202, 45, 151, 5, 84, 87,
			148, 86, 143, 159, 75, 80, 48,
			52, 9, 21, 73, 152, 91, 210, 181,
			160, 79, 84, 131, 166, 199, 46, 67,
			161, 152, 168, 35, 100, 202, 103, 226,
			171, 34, 125, 81, 170, 9, 30, 223,
			148, 235, 137, 7, 53, 105, 156, 152,
			114, 94, 206, 106, 27, 12, 129, 27,
			42, 210, 50, 175, 4, 213, 194, 191,
			5, 223, 253, 216,
		}

		yetAnotherPacket := []byte{
			255, 255, 255, 255, 192, 116, 60, 67,
			77, 121, 75, 18, 231, 8, 0, 0,
			125, 102, 87, 13, 63, 21, 230, 174,
			113, 105, 84, 119, 15, 212, 81,
			181, 18, 115, 9, 39, 55, 226, 36,
			70, 59, 176, 242, 67, 120, 17, 75,
			43,
		}
	*/

	fromNode := binary.LittleEndian.Uint32(packet[4:8])
	packetId := binary.LittleEndian.Uint32(packet[8:12])

	nonce := make([]byte, 16)
	binary.LittleEndian.PutUint64(nonce[0:8], uint64(packetId))
	binary.LittleEndian.PutUint32(nonce[8:12], fromNode)

	encryptedPayload := packet[16:]

	block, err := aes.NewCipher(defaultPublicKey)
	assert.NoError(t, err)

	stream := cipher.NewCTR(block, nonce)
	decrypted := make([]byte, len(encryptedPayload))
	stream.XORKeyStream(decrypted, encryptedPayload)

	t.Logf("Decrypted: %v %q", decrypted, string(decrypted))

	firstByte := decrypted[0]
	t.Logf("First byte: 0x%02X (binary: %08b)", firstByte, firstByte)
	fieldNum := firstByte >> 3
	wireType := firstByte & 0x07
	t.Logf("Field number: %d, Wire type: %d\n", fieldNum, wireType)

	meshPacket := pb.Data{}
	err = proto.Unmarshal(decrypted, &meshPacket)
	assert.NoError(t, err)

	t.Logf("Portnum: %v\n", meshPacket.Portnum)
	t.Logf("Payload: %v\n", meshPacket.Payload)

	if meshPacket.Portnum == pb.PortNum_TEXT_MESSAGE_APP {
		t.Logf("Text message: %s\n", string(meshPacket.Payload))
	}
}

func TestReceivedPacketDecode(t *testing.T) {
	packet := client.PacketReceived{
		PacketRSSI_dBm: 0,
		PacketSNR_dB:   0,
		SignalRSSI_dBm: 0,
		Data: []byte{
			0xff, 0xff, 0xff, 0xff,
			0x44, 0x33, 0x22, 0x11,
			0x5f, 0xb1, 0x3e, 0xfb,
			0xe7,       // Flags
			0x08,       // Hash
			0x00, 0x00, // Align
			0x7d, 0x7f, 0xa9, 0x49, 0x1a, 0xd1, 0x39, 0xf4,
			0xf9, 0xf3, 0x57, 0x5b, 0x83, 0x05, 0x4d, 0xa6,
			0xdb, 0x2c, 0x25, 0xa8, 0x82, 0x25, 0x5f, 0xa4,
			0x7e, 0x91, 0x9f, 0xff, 0x39,
		},
	}

	channel := NewChannel(0, "LongFast", defaultPublicKey)
	meshPacket, err := channel.DecodePacket(&packet)
	assert.NoError(t, err)

	assert.Equal(t, uint32(0xFFFFFFFF), meshPacket.To)
	assert.Equal(t, uint32(0x11223344), meshPacket.From)
	assert.Equal(t, uint32(0xFb3EB15F), meshPacket.Id)
	assert.Equal(t, uint32(7), meshPacket.HopLimit)
	assert.Equal(t, uint32(7), meshPacket.HopStart)

	decoded, ok := meshPacket.PayloadVariant.(*pb.MeshPacket_Decoded)
	assert.True(t, ok)

	assert.Equal(t, pb.PortNum_TEXT_MESSAGE_APP, decoded.Decoded.Portnum)
	assert.Equal(t, "Hello from Waveshare USB!", string(decoded.Decoded.Payload))
}

/*
Private message:
ea2ffd25	To 			25fd2fea
68643c43	From		433c6468
3c572524	Packet ID	2425573c
ef			Flags		WantAck is set
00			Hash
0068
2ce70ee8a8d5e651743013beb3324878ebf42df45158
*/

/*
Packet from Heltec V3:

ffffffff	To
68643c43	From
73e66127	Packet Id
e7			Flasg
08			Hash
0068
d40acbe7d253ff92ec
*/
