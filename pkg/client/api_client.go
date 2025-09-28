package client

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

type ApiClient struct {
	serial *SerialClient
	ctx    context.Context
	cancel context.CancelFunc
}

func NewApiClient() *ApiClient {

	client := &ApiClient{
		serial: NewSerialClient(),
		ctx:    nil,
		cancel: nil,
	}

	return client
}

func (self *ApiClient) Open(portName string) error {
	if self.serial.IsOpen() {
		return fmt.Errorf("serial port already open")
	}

	err := self.serial.Open(portName)

	if err != nil {
		return err
	}

	self.ctx, self.cancel = context.WithCancel(context.Background())

	// Receive data from device
	go func() {
		for {
			select {
			case <-self.ctx.Done():
				return
			default:
				msg, err := self.serial.ReceiveMessage()

				if err != nil {
					_, ok := err.(*TimeoutError)
					if ok {
						// Timeout - keep reading
						continue

					}

					// Failure - terminate reading loop
					return
				}

				err = self.handleMessage(msg)
				if err != nil {
					// Error handling message (invalid message?)
					continue
				}
			}
		}
	}()

	return nil
}

func (self *ApiClient) Close() error {
	self.cancel()

	if self.serial.IsOpen() {
		return self.serial.Close()
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
