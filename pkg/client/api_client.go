package client

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

type ApiClient struct {
	serial *SerialClient
	ctx    context.Context
	cancel context.CancelFunc

	send chan ApiMessage
	recv chan ApiMessage
}

func NewApiClient() *ApiClient {

	client := &ApiClient{
		serial: NewSerialClient(),
		ctx:    nil,
		cancel: nil,
		send:   make(chan ApiMessage, 1),
		recv:   make(chan ApiMessage, 1),
	}

	return client
}

func (c *ApiClient) Open(portName string) error {
	if c.serial.IsOpen() {
		return fmt.Errorf("serial port already open")
	}

	err := c.serial.Open(portName)

	if err != nil {
		return err
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())

	// Send data to device
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			case msg := <-c.send:
				err = c.sendMessage(msg)

				if err != nil {
					// @todo Report error
				}
			}
		}
	}()

	// Receive data from device
	go func() {
		for {
			select {
			case <-c.ctx.Done():
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
	c.cancel()

	if c.serial.IsOpen() {
		return c.serial.Close()
	}

	return nil
}

func (c *ApiClient) sendMessage(msg ApiMessage) error {
	message := msg.SerializeRequest()

	err := c.serial.SendMessage(&message)
	if err != nil {
		return err
	}

	return nil
}

func (c *ApiClient) handleMessage(message *Message) error {
	var msg ApiMessage = nil

	switch message.Type {
	case MSG_VERSION:
		msg = &Version{}
	case MSG_LORA_PARAMS:
		msg = &LoRaParameters{}
	case MSG_LORA_PACKET:
		msg = &LoRaPacketParameters{}
	case MSG_RX_PARAMS:
		msg = &RxParameters{}
	case MSG_TX_PARAMS:
		msg = &TxParameters{}
	case MSG_FREQUENCY:
		msg = &RadioFrequency{}
	case MSG_FALLBACK_MODE:
		msg = &RxTxFallbackMode{}
	case MSG_RSSI:
		msg = &InstantaneousRSSI{}
	case MSG_RX:
		msg = &SwitchToRx{}
	case MSG_TX:
		msg = &Transmit{}
	case MSG_STANDBY:
		msg = &Standby{}
	case MSG_TIMEOUT:
		msg = &RxTxTimeout{}
	case MSG_PACKET_RECEIVED:
		msg = &PacketReceived{}
	case MSG_PACKET_TRANSMITTED:
		msg = &PacketTransmitted{}
	case MSG_CONTINUOUS_RSSI:
		msg = &ContinuoisRSSI{}
	default:
		return fmt.Errorf("invalid message received from device")
	}

	if err := msg.DeserializeResponse(message); err != nil {
		return nil
	}

	if msg != nil {
		c.recv <- msg
	}

	return nil
}

func (c *ApiClient) SendMessage(msg ApiMessage) {
	c.send <- msg
}

func (c *ApiClient) SendRequest(msg ApiMessage, timeout time.Duration) ApiMessage {
	c.sendMessage(msg)

	start := time.Now()

	for {
		elapsed := time.Since(start)
		if elapsed >= timeout {
			return nil
		}

		select {
		case res := <-c.recv:
			if reflect.TypeOf(res) == reflect.TypeOf(msg) {
				return res
			}
		case <-time.After(timeout - elapsed):
			return nil
		}
	}
}

func (c *ApiClient) ReceiveMessage(timeout time.Duration) ApiMessage {
	select {
	case msg := <-c.recv:
		return msg
	case <-time.After(timeout):
		return nil
	}
}
