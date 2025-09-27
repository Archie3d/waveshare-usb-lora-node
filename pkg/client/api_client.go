package client

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

const (
	// Messages to device
	MSG_INVALID           = 0x00
	MSG_GET_VERSION       = 0x01
	MSG_SET_LORA_PARAMS   = 0x02
	MSG_SET_LORA_PACKET   = 0x03
	MSG_SET_RX_PARAMS     = 0x04
	MSG_SET_TX_PARAMS     = 0x05
	MSG_SET_FREQUENCY     = 0x06
	MSG_SET_FALLBACK_MODE = 0x07
	MSG_GET_RSSI          = 0x08
	MSG_SET_RX            = 0x09
	MSG_SET_TX            = 0x0A
	MSG_SET_STANDBY       = 0x0B

	// Response messages from device to controller
	MSG_VERSION       = 0x81
	MSG_LORA_PARAMS   = 0x82
	MSG_LORA_PACKET   = 0x83
	MSG_RX_PARAMS     = 0x84
	MSG_TX_PARAMS     = 0x85
	MSG_FREQUENCY     = 0x86
	MSG_FALLBACK_MODE = 0x87
	MSG_RSSI          = 0x88
	MSG_RX            = 0x89
	MSG_TX            = 0x8A
	MSG_STANDBY       = 0x8B

	// Unsolicited messages (from device)
	MSG_TIMEOUT            = 0x90
	MSG_PACKET_RECEIVED    = 0x91
	MSG_PACKET_TRANSMITTED = 0x92
	MSG_CONTINUOUS_RSSI    = 0x93
	MSG_LOGGING            = 0x9F
)

// LoRa spreading factors
const (
	LORA_SF5  = 0x05
	LORA_SF6  = 0x06
	LORA_SF7  = 0x07
	LORA_SF8  = 0x08
	LORA_SF9  = 0x09
	LORA_SF10 = 0x0A
	LORA_SF11 = 0x0B
	LORA_SF12 = 0x0C
)

// LoRa bandwidths
const (
	LORA_BW_500 = 6
	LORA_BW_250 = 5
	LORA_BW_125 = 4
	LORA_BW_062 = 3
	LORA_BW_041 = 10
	LORA_BW_031 = 2
	LORA_BW_020 = 9
	LORA_BW_015 = 1
	LORA_BW_010 = 8
	LORA_BW_007 = 0
)

// LoRa coding rates
const (
	LORA_CR_4_5 = 0x01
	LORA_CR_4_6 = 0x02
	LORA_CR_4_7 = 0x03
	LORA_CR_4_8 = 0x04
)

// Power ramp
const (
	POWER_RAMP_10   = 0x00
	POWER_RAMP_20   = 0x01
	POWER_RAMP_40   = 0x02
	POWER_RAMP_80   = 0x03
	POWER_RAMP_200  = 0x04
	POWER_RAMP_800  = 0x05
	POWER_RAMP_1700 = 0x06
	POWER_RAMP_3400 = 0x07
)

// Fallback modes
const (
	FALLBACK_STANDBY_RC      = 0x20
	FALLBACK_STANDBY_XOSC    = 0x30
	FALLBACK_STANDBY_XOSC_RX = 0x31
	FALLBACK_FS              = 0x40
)

// Standby modes
const (
	STANDBY_RC   = 0x00
	STANDBY_XOSC = 0x01
)

type Version struct {
	Major byte
	Minor byte
	Patch byte
}

type LoRaParameters struct {
	SpreadingFactor byte
	Bandwidth       byte
	CodingRate      byte
	LowDataRate     bool
}

type LoRaPacketParameters struct {
	PreambleLength uint16
	ImplicitHeader bool
	SyncWord       byte
	CrcOn          bool
	InvertIQ       bool
}

type RxParameters struct {
	RxBoost bool
}

type TxParameters struct {
	DutyCycle byte
	HpMax     byte
	Power     byte
	RampTime  byte
}

type ApiClient struct {
	serial       *SerialClient
	Timeout      bool
	RxTxComplete bool
	RSSI         int16

	quit    chan bool
	version chan Version
}

func NewApiClient() *ApiClient {
	client := &ApiClient{
		serial:  NewSerialClient(),
		quit:    make(chan bool),
		version: make(chan Version),
	}

	return client
}

func (c *ApiClient) Open(portName string) error {
	if c.serial.IsOpen() {
		return errors.New("serial port already open")
	}

	err := c.serial.Open(portName)

	if err != nil {
		return err
	}

	go func() {
		fmt.Println("Starting message handler")

		for {
			select {
			case <-c.quit:
				return
			default:
				msg, err := c.serial.ReceiveMessage()
				if err != nil {

					_, ok := err.(*TimeoutError)
					if ok {
						// Timeout - keep reading
						continue
					}

					// Failure - terminate reading loop
					return
				}

				err = c.handleMessage(msg)
				if err != nil {
					// Error handling message (invalid message?)
					continue
				}
			}
		}
	}()

	return nil
}

func (c *ApiClient) Close() error {
	if c.serial.IsOpen() {
		c.quit <- true
		return c.serial.Close()
	}

	return nil
}

func (c *ApiClient) handleMessage(message *Message) error {
	switch message.Type {
	case MSG_VERSION:
		c.version <- Version{
			Major: message.Payload[0],
			Minor: message.Payload[1],
			Patch: message.Payload[2],
		}
	case MSG_TIMEOUT:
		c.Timeout = true
		fmt.Print("Timeout\n")
	case MSG_PACKET_RECEIVED:
		c.RxTxComplete = true
		fmt.Printf("Received packet: %v\n", message.Payload)
	case MSG_PACKET_TRANSMITTED:
		c.RxTxComplete = true
		fmt.Printf("Transmitted packet: %v\n", message.Payload)
	case MSG_CONTINUOUS_RSSI:
		c.RSSI = int16(binary.LittleEndian.Uint16(message.Payload))
		fmt.Printf("RSSI: %v\n", c.RSSI)
	case MSG_LOGGING:
		fmt.Printf("Logging: %v\n", string(message.Payload))
	default:
		fmt.Printf("Received invalid message: %v\n", message)
	}

	return nil
}

func (c *ApiClient) waitForResponse(messageType byte) (*Message, error) {

	for {
		message, err := c.serial.ReceiveMessage()

		if err != nil {
			return nil, err
		}

		if message.Type == messageType {
			return message, nil
		}

		err = c.handleMessage(message)
		if err != nil {
			return nil, err
		}
	}
}

func (c *ApiClient) Wait(timeout uint32) error {
	// Starting time
	startTime := time.Now()

	for time.Since(startTime) < time.Duration(timeout)*time.Millisecond {
		msg, _ := c.serial.ReceiveMessage()

		if msg != nil {
			err := c.handleMessage(msg)
			if err != nil {
				return err
			}

			if c.Timeout || c.RxTxComplete {
				break
			}
		}
	}

	return nil
}

func (c *ApiClient) GetVersion() (Version, error) {
	message := &Message{
		Type:    MSG_GET_VERSION,
		Payload: []byte{},
	}

	err := c.serial.SendMessage(message)
	if err != nil {
		return Version{}, err
	}

	version := <-c.version
	/*
		resp, err := c.waitForResponse(MSG_VERSION)
		if err != nil {
			return Version{}, err
		}

		version := Version{
			Major: resp.Payload[0],
			Minor: resp.Payload[1],
			Patch: resp.Payload[2],
		}
	*/
	return version, nil
}

func (c *ApiClient) SetLoRaParameters(params *LoRaParameters) error {
	message := &Message{
		Type:    MSG_SET_LORA_PARAMS,
		Payload: []byte{},
	}

	message.Payload = append(message.Payload, params.SpreadingFactor)
	message.Payload = append(message.Payload, params.Bandwidth)
	message.Payload = append(message.Payload, params.CodingRate)
	if params.LowDataRate {
		message.Payload = append(message.Payload, 0x01)
	} else {
		message.Payload = append(message.Payload, 0x00)
	}

	err := c.serial.SendMessage(message)
	if err != nil {
		return err
	}

	resp, err := c.waitForResponse(MSG_LORA_PARAMS)
	if err != nil {
		return err
	}

	if len(resp.Payload) != 4 {
		return errors.New("invalid response")
	}

	return nil
}

func (c *ApiClient) SetLoRaPacketParameters(params *LoRaPacketParameters) error {
	message := &Message{
		Type:    MSG_SET_LORA_PACKET,
		Payload: []byte{},
	}

	binary.LittleEndian.PutUint16(message.Payload, params.PreambleLength)

	if params.ImplicitHeader {
		message.Payload = append(message.Payload, 0x01)
	} else {
		message.Payload = append(message.Payload, 0x00)
	}

	message.Payload = append(message.Payload, params.SyncWord)

	if params.CrcOn {
		message.Payload = append(message.Payload, 0x01)
	} else {
		message.Payload = append(message.Payload, 0x00)
	}

	if params.InvertIQ {
		message.Payload = append(message.Payload, 0x01)
	} else {
		message.Payload = append(message.Payload, 0x00)
	}

	err := c.serial.SendMessage(message)
	if err != nil {
		return err
	}

	resp, err := c.waitForResponse(MSG_LORA_PACKET)
	if err != nil {
		return err
	}

	if len(resp.Payload) != 6 {
		return errors.New("invalid response")
	}

	return nil
}

func (c *ApiClient) SetRxParameters(params *RxParameters) error {
	message := &Message{
		Type:    MSG_SET_RX_PARAMS,
		Payload: []byte{},
	}

	if params.RxBoost {
		message.Payload = append(message.Payload, 0x01)
	} else {
		message.Payload = append(message.Payload, 0x00)
	}

	err := c.serial.SendMessage(message)
	if err != nil {
		return err
	}

	c.Timeout = false

	resp, err := c.waitForResponse(MSG_RX_PARAMS)
	if err != nil {
		return err
	}

	if len(resp.Payload) != 1 {
		return errors.New("invalid response")
	}

	return nil
}

func (c *ApiClient) SetTxParameters(params *TxParameters) error {
	message := &Message{
		Type:    MSG_SET_TX_PARAMS,
		Payload: []byte{},
	}

	message.Payload = append(message.Payload, params.DutyCycle)
	message.Payload = append(message.Payload, params.HpMax)
	message.Payload = append(message.Payload, params.Power)
	message.Payload = append(message.Payload, params.RampTime)

	err := c.serial.SendMessage(message)
	if err != nil {
		return err
	}

	resp, err := c.waitForResponse(MSG_TX_PARAMS)
	if err != nil {
		return err
	}

	if len(resp.Payload) != 4 {
		return errors.New("invalid response")
	}

	return nil
}

func (c *ApiClient) SetFrequency(frequency uint32) error {
	message := &Message{
		Type:    MSG_SET_FREQUENCY,
		Payload: []byte{},
	}

	frequencyBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(frequencyBytes, frequency)
	message.Payload = append(message.Payload, frequencyBytes...)

	err := c.serial.SendMessage(message)
	if err != nil {
		return err
	}

	resp, err := c.waitForResponse(MSG_FREQUENCY)
	if err != nil {
		return err
	}

	if len(resp.Payload) != 4 {
		return errors.New("invalid response")
	}

	return nil
}

func (c *ApiClient) SetFallbackMode(mode byte) error {
	message := &Message{
		Type:    MSG_SET_FALLBACK_MODE,
		Payload: []byte{mode},
	}

	err := c.serial.SendMessage(message)
	if err != nil {
		return err
	}

	resp, err := c.waitForResponse(MSG_FALLBACK_MODE)
	if err != nil {
		return err
	}

	if len(resp.Payload) != 1 {
		return errors.New("invalid response")
	}

	return nil
}

func (c *ApiClient) GetRSSI() (int16, error) {
	message := &Message{
		Type:    MSG_GET_RSSI,
		Payload: []byte{},
	}

	err := c.serial.SendMessage(message)
	if err != nil {
		return 0, err
	}

	resp, err := c.waitForResponse(MSG_RSSI)
	if err != nil {
		return 0, err
	}

	if len(resp.Payload) != 2 {
		return 0, errors.New("invalid response")
	}

	rssi := int16(binary.LittleEndian.Uint16(resp.Payload))

	return rssi, nil
}

func (c *ApiClient) SetRx(timeout uint32, reportRSSI bool) error {
	message := &Message{
		Type:    MSG_SET_RX,
		Payload: []byte{},
	}

	timeoutBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(timeoutBytes, timeout)
	message.Payload = append(message.Payload, timeoutBytes...)

	if reportRSSI {
		message.Payload = append(message.Payload, 0x01)
	} else {
		message.Payload = append(message.Payload, 0x00)
	}

	err := c.serial.SendMessage(message)
	if err != nil {
		return err
	}

	c.Timeout = false
	c.RxTxComplete = false

	resp, err := c.waitForResponse(MSG_RX)
	if err != nil {
		return err
	}

	if len(resp.Payload) != 5 {
		return errors.New("invalid response")
	}

	return nil
}
