package meshtastic

import (
	"context"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/client"
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

type PacketTimespamp struct {
	dest     uint32
	from     uint32
	id       uint32
	received time.Time
}

type MeshtasticClient struct {
	apiClient *client.ApiClient
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	rssi_dBm atomic.Int32

	seenPackets []PacketTimespamp

	Packets chan *client.PacketReceived
}

// Create a new Meshtastic client but do not run it yet.
func NewMeshtasticClient() *MeshtasticClient {
	return &MeshtasticClient{
		apiClient: client.NewApiClient(),
		Packets:   make(chan *client.PacketReceived, 10),
	}
}

// Open serial port to talk to the LoRa device and
// start receiving Meshtastic messages.
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

// Stop processing messages and close the serial port to the device.
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
		// Switch back to RX mode
		c.switchToRx()

		// Purge records of older packets
		c.forgetOldSeenPackets()

		record := PacketTimespamp{
			dest:     binary.BigEndian.Uint32(packet.Data[0:4]),
			from:     binary.BigEndian.Uint32(packet.Data[4:8]),
			id:       binary.LittleEndian.Uint32(packet.Data[8:12]),
			received: time.Now(),
		}

		if !c.haveSeenPacket(&record) {
			c.seenPackets = append(c.seenPackets, record)

			c.Packets <- packet
		}

	} else if rssi, ok := msg.(*client.ContinuoisRSSI); ok {
		// Capture RSSI
		c.rssi_dBm.Store(int32(rssi.RSSI_dBm))
	}
}

func (c *MeshtasticClient) forgetOldSeenPackets() {
	now := time.Now()

	var seenPackets []PacketTimespamp

	for _, sp := range c.seenPackets {
		if now.Sub(sp.received) < 30*time.Second {
			seenPackets = append(seenPackets, sp)
		}
	}

	c.seenPackets = seenPackets
}

func (c *MeshtasticClient) haveSeenPacket(p *PacketTimespamp) bool {
	for _, sp := range c.seenPackets {
		if sp.dest == p.dest && sp.from == p.from && sp.id == p.id {
			return true
		}
	}

	return false
}
