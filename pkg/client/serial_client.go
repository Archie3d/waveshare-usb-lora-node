package client

import (
	"fmt"
	"time"

	"go.bug.st/serial"
)

const (
	DEFAULT_BAUD_RATE = 115200

	START         = 0xAA
	ESCAPE        = 0x7D
	ESCAPE_START  = 0x8A
	ESCAPE_ESCAPE = 0x5D
)

func crc16(crc0 uint16, data []byte) uint16 {
	crc := crc0
	for _, b := range data {
		a := (crc >> 8) ^ uint16(b)
		crc = (a << 2) ^ (a << 1) ^ a ^ (crc << 8)
	}
	return crc
}

func escape(data []byte) []byte {
	escaped := make([]byte, 0)

	for _, b := range data {
		switch b {
		case START:
			escaped = append(escaped, ESCAPE)
			escaped = append(escaped, ESCAPE_START)
		case ESCAPE:
			escaped = append(escaped, ESCAPE)
			escaped = append(escaped, ESCAPE_ESCAPE)
		default:
			escaped = append(escaped, b)
		}
	}
	return escaped
}

func unescape(data []byte) []byte {
	unescaped := make([]byte, 0)

	for i := 0; i < len(data); i++ {
		if data[i] == ESCAPE {
			if data[i+1] == ESCAPE_START {
				unescaped = append(unescaped, START)
			} else if data[i+1] == ESCAPE_ESCAPE {
				unescaped = append(unescaped, ESCAPE)
			}
			i++
		} else {
			unescaped = append(unescaped, data[i])
		}
	}
	return unescaped
}

type TimeoutError struct{}

func (e *TimeoutError) Error() string {
	return "timeout"
}

type Message struct {
	Type    byte
	Payload []byte
}

type SerialClient struct {
	port serial.Port
}

func NewSerialClient() *SerialClient {

	client := &SerialClient{}
	client.port = nil

	return client
}

func (c *SerialClient) Open(portName string) error {
	mode := &serial.Mode{
		BaudRate: DEFAULT_BAUD_RATE,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return err
	}

	var timeout time.Duration = 1 * time.Second
	port.SetReadTimeout(timeout)

	c.port = port
	return nil
}

func (c *SerialClient) Close() error {
	if c.port == nil {
		return nil
	}

	err := c.port.Close()
	if err != nil {
		return err
	}

	c.port = nil
	return nil
}

func (c *SerialClient) IsOpen() bool {
	return c.port != nil
}

func (c *SerialClient) send(data []byte) error {
	if c.port == nil {
		return nil
	}

	// Escape the data first
	data = escape(data)

	packet := make([]byte, 1)
	packet[0] = START

	packet = append(packet, data...)

	n, err := c.port.Write(packet)
	if err != nil {
		return err
	}

	if n < 1 {
		return &TimeoutError{}
	}

	return nil
}

func (c *SerialClient) recv_byte() (byte, error) {
	// Return error if port is not open
	if c.port == nil {
		return 0, fmt.Errorf("port is not open")
	}

	buf := make([]byte, 1)
	n, err := c.port.Read(buf)

	if err != nil {
		return 0, err
	}

	if n != 1 {
		return 0, &TimeoutError{}
	}

	if buf[0] != ESCAPE {
		return buf[0], nil
	}

	// Read the next byte for the escape sequence
	n, err = c.port.Read(buf)
	if err != nil {
		return 0, err
	}

	if n < 1 {
		return 0, fmt.Errorf("incomplete escape sequence")
	}

	switch buf[0] {
	case ESCAPE_START:
		return START, nil
	case ESCAPE_ESCAPE:
		return ESCAPE, nil
	}

	return 0, fmt.Errorf("invalid escape sequence")
}

/*
Send an unstructured message to the serial port.
*/
func (c *SerialClient) SendMessage(message *Message) error {
	if c.port == nil {
		return fmt.Errorf("port is not open")
	}

	data := make([]byte, 0)

	data = append(data, message.Type)

	payloadLength := uint16(len(message.Payload))
	data = append(data, byte(payloadLength&0xFF))
	data = append(data, byte(payloadLength>>8))

	data = append(data, message.Payload...)

	crc := crc16(0, data)
	data = append(data, byte(crc&0xFF))
	data = append(data, byte(crc>>8))

	err := c.send(data)
	if err != nil {
		return err
	}

	return nil
}

/*
Receive an unstructured message from the serial port.
*/
func (c *SerialClient) ReceiveMessage() (*Message, error) {
	if c.port == nil {
		return nil, fmt.Errorf("port is not open")
	}

	startDetected := false

	for !startDetected {
		b, err := c.recv_byte()
		if err != nil {
			return nil, err
		}

		if b == START {
			startDetected = true
		}
	}

	calculatedCrc := uint16(0)

	messageType, err := c.recv_byte()
	if err != nil {
		return nil, err
	}

	calculatedCrc = crc16(calculatedCrc, []byte{messageType})

	var payloadLengthLsb byte = 0
	var payloadLengthMsb byte = 0

	payloadLengthLsb, err = c.recv_byte()
	if err != nil {
		return nil, err
	}
	calculatedCrc = crc16(calculatedCrc, []byte{payloadLengthLsb})

	payloadLengthMsb, err = c.recv_byte()
	if err != nil {
		return nil, err
	}
	calculatedCrc = crc16(calculatedCrc, []byte{payloadLengthMsb})

	payloadLength := uint16(payloadLengthMsb)<<8 | uint16(payloadLengthLsb)

	var received uint16 = 0
	payload := make([]byte, payloadLength)

	for received < payloadLength {
		b, err := c.recv_byte()
		if err != nil {
			return nil, err
		}

		payload[received] = b
		received++
	}

	calculatedCrc = crc16(calculatedCrc, payload)

	var crcLsb byte = 0
	var crcMsb byte = 0

	crcLsb, err = c.recv_byte()
	if err != nil {
		return nil, err
	}

	crcMsb, err = c.recv_byte()
	if err != nil {
		return nil, err
	}

	crc := uint16(crcMsb)<<8 | uint16(crcLsb)

	if crc != calculatedCrc {
		return nil, fmt.Errorf("CRC mismatch")
	}

	return &Message{
		Type:    messageType,
		Payload: payload,
	}, nil
}
