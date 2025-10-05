package client

import (
	"encoding/binary"
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

type ApiMessage interface {
	SerializeRequest() Message
	DeserializeResponse(msg *Message) error
}

func boolToByte(b bool) byte {
	if b {
		return 0x01
	}

	return 0x00
}

func byteToBool(b byte) bool {
	return b != 0x00
}

type MessageTypeError struct{}

func (e *MessageTypeError) Error() string {
	return "invalid message type"
}

type MessagePayloadSizeError struct{}

func (e *MessagePayloadSizeError) Error() string {
	return "invalid message payload size"
}

//------------------------------------------------------------------------------

type Version struct {
	Major byte
	Minor byte
	Patch byte
}

func (m *Version) SerializeRequest() Message {
	return Message{
		Type:    MSG_GET_VERSION,
		Payload: []byte{},
	}
}

func (m *Version) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_VERSION {
		return &MessageTypeError{}
	}

	if len(msg.Payload) != 3 {
		return &MessagePayloadSizeError{}
	}

	m.Major = msg.Payload[0]
	m.Minor = msg.Payload[1]
	m.Patch = msg.Payload[2]

	return nil
}

//------------------------------------------------------------------------------

type LoRaParameters struct {
	SpreadingFactor byte
	Bandwidth       byte
	CodingRate      byte
	LowDataRate     bool
}

func (m *LoRaParameters) SerializeRequest() Message {
	return Message{
		Type: MSG_SET_LORA_PARAMS,
		Payload: []byte{
			m.SpreadingFactor,
			m.Bandwidth,
			m.CodingRate,
			boolToByte(m.LowDataRate),
		},
	}
}

func (m *LoRaParameters) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_LORA_PARAMS {
		return &MessageTypeError{}
	}

	if len(msg.Payload) != 4 {
		return &MessagePayloadSizeError{}
	}

	m.SpreadingFactor = msg.Payload[0]
	m.Bandwidth = msg.Payload[1]
	m.CodingRate = msg.Payload[2]
	m.LowDataRate = false

	if msg.Payload[3] != 0x00 {
		m.LowDataRate = true
	}

	return nil
}

//------------------------------------------------------------------------------

type LoRaPacketParameters struct {
	PreambleLength uint16
	ImplicitHeader bool
	SyncWord       byte
	CrcOn          bool
	InvertIQ       bool
}

func (m *LoRaPacketParameters) SerializeRequest() Message {
	message := Message{
		Type:    MSG_SET_LORA_PACKET,
		Payload: make([]byte, 6),
	}

	binary.LittleEndian.PutUint16(message.Payload[0:2], m.PreambleLength)
	message.Payload[2] = boolToByte(m.ImplicitHeader)
	message.Payload[3] = m.SyncWord
	message.Payload[4] = boolToByte(m.CrcOn)
	message.Payload[5] = boolToByte(m.InvertIQ)

	return message
}

func (m *LoRaPacketParameters) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_LORA_PACKET {
		return &MessageTypeError{}
	}

	if len(msg.Payload) != 6 {
		return &MessagePayloadSizeError{}
	}

	m.PreambleLength = binary.LittleEndian.Uint16(msg.Payload[0:2])
	m.ImplicitHeader = byteToBool(msg.Payload[2])
	m.SyncWord = msg.Payload[3]
	m.CrcOn = byteToBool(msg.Payload[4])
	m.InvertIQ = byteToBool(msg.Payload[5])

	return nil
}

//------------------------------------------------------------------------------

type RxParameters struct {
	RxBoost bool
}

func (m *RxParameters) SerializeRequest() Message {
	return Message{
		Type:    MSG_SET_RX_PARAMS,
		Payload: []byte{boolToByte(m.RxBoost)},
	}
}

func (m *RxParameters) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_RX_PARAMS {
		return &MessageTypeError{}
	}

	if len(msg.Payload) != 1 {
		return &MessagePayloadSizeError{}
	}

	m.RxBoost = byteToBool(msg.Payload[0])

	return nil
}

//------------------------------------------------------------------------------

type TxParameters struct {
	DutyCycle byte
	HpMax     byte
	Power     byte
	RampTime  byte
}

func (m *TxParameters) SerializeRequest() Message {
	return Message{
		Type: MSG_SET_TX_PARAMS,
		Payload: []byte{
			m.DutyCycle,
			m.HpMax,
			m.Power,
			m.RampTime,
		},
	}
}

func (m *TxParameters) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_TX_PARAMS {
		return &MessageTypeError{}
	}

	if len(msg.Payload) != 4 {
		return &MessagePayloadSizeError{}
	}

	m.DutyCycle = msg.Payload[0]
	m.HpMax = msg.Payload[1]
	m.Power = msg.Payload[2]
	m.RampTime = msg.Payload[3]

	return nil
}

//------------------------------------------------------------------------------

type RadioFrequency struct {
	Frequency_Hz uint32
}

func (m *RadioFrequency) SerializeRequest() Message {
	message := Message{
		Type:    MSG_SET_FREQUENCY,
		Payload: make([]byte, 4),
	}

	binary.LittleEndian.PutUint32(message.Payload, m.Frequency_Hz)

	return message
}

func (m *RadioFrequency) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_FREQUENCY {
		return &MessageTypeError{}
	}

	if len(msg.Payload) != 4 {
		return &MessagePayloadSizeError{}
	}

	m.Frequency_Hz = binary.LittleEndian.Uint32(msg.Payload)

	return nil
}

//------------------------------------------------------------------------------

type RxTxFallbackMode struct {
	FallbackMode byte
}

func (m *RxTxFallbackMode) SerializeRequest() Message {
	return Message{
		Type:    MSG_SET_FALLBACK_MODE,
		Payload: []byte{m.FallbackMode},
	}
}

func (m *RxTxFallbackMode) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_FALLBACK_MODE {
		return &MessageTypeError{}
	}

	if len(msg.Payload) != 1 {
		return &MessagePayloadSizeError{}
	}

	m.FallbackMode = msg.Payload[0]

	return nil
}

//------------------------------------------------------------------------------

type InstantaneousRSSI struct {
	RSSI_dBm int16
}

func (m *InstantaneousRSSI) SerializeRequest() Message {
	return Message{
		Type:    MSG_GET_RSSI,
		Payload: []byte{},
	}
}

func (m *InstantaneousRSSI) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_RSSI {
		return &MessageTypeError{}
	}

	if len(msg.Payload) != 2 {
		return &MessagePayloadSizeError{}
	}

	m.RSSI_dBm = int16(binary.LittleEndian.Uint16(msg.Payload))

	return nil
}

//------------------------------------------------------------------------------

type SwitchToRx struct {
	Timeout_ms           uint32
	EnableContinuousRSSI bool
}

func (m *SwitchToRx) SerializeRequest() Message {
	message := Message{
		Type:    MSG_SET_RX,
		Payload: make([]byte, 5),
	}

	binary.LittleEndian.PutUint32(message.Payload[0:4], m.Timeout_ms)
	message.Payload[4] = boolToByte(m.EnableContinuousRSSI)

	return message
}

func (m *SwitchToRx) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_RX {
		return &MessageTypeError{}
	}

	if len(msg.Payload) != 5 {
		return &MessagePayloadSizeError{}
	}

	m.Timeout_ms = binary.LittleEndian.Uint32(msg.Payload[0:4])
	m.EnableContinuousRSSI = byteToBool(msg.Payload[4])

	return nil
}

//------------------------------------------------------------------------------

type Transmit struct {
	Timeout_ms uint32 // Used in request only
	Data       []byte // Used in request only
	Busy       bool   // Used in response only
}

func (m *Transmit) SerializeRequest() Message {
	message := Message{
		Type:    MSG_SET_TX,
		Payload: make([]byte, len(m.Data)+4),
	}

	binary.LittleEndian.PutUint32(message.Payload[0:4], m.Timeout_ms)
	copy(message.Payload[4:], m.Data)

	return message
}

func (m *Transmit) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_TX {
		return &MessageTypeError{}
	}

	if len(msg.Payload) != 1 {
		return &MessagePayloadSizeError{}
	}

	m.Busy = byteToBool(msg.Payload[0])

	return nil
}

//------------------------------------------------------------------------------

type Standby struct {
	StandbyMode byte
}

func (m *Standby) SerializeRequest() Message {
	return Message{
		Type:    MSG_SET_STANDBY,
		Payload: []byte{m.StandbyMode},
	}
}

func (m *Standby) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_STANDBY {
		return &MessageTypeError{}
	}

	if len(msg.Payload) != 1 {
		return &MessagePayloadSizeError{}
	}

	m.StandbyMode = msg.Payload[0]

	return nil
}

//==============================================================================
/*
Unsolicited messages from the device
*/

type RxTxTimeout struct {
}

func (m *RxTxTimeout) SerializeRequest() Message {
	return Message{
		Type:    MSG_TIMEOUT,
		Payload: []byte{},
	}
}

func (m *RxTxTimeout) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_TIMEOUT {
		return &MessageTypeError{}
	}

	if len(msg.Payload) != 0 {
		return &MessagePayloadSizeError{}
	}

	return nil
}

//------------------------------------------------------------------------------

type PacketReceived struct {
	PacketRSSI_dBm int8
	PacketSNR_dB   int8
	SignalRSSI_dBm int8
	Data           []byte
}

func (m *PacketReceived) SerializeRequest() Message {
	return Message{
		Type:    MSG_PACKET_RECEIVED,
		Payload: make([]byte, 3+len(m.Data)),
	}
}

func (m *PacketReceived) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_PACKET_RECEIVED {
		return &MessageTypeError{}
	}

	if len(msg.Payload) < 3 {
		return &MessagePayloadSizeError{}
	}

	m.PacketRSSI_dBm = int8(msg.Payload[0])
	m.PacketSNR_dB = int8(msg.Payload[1])
	m.SignalRSSI_dBm = int8(msg.Payload[2])

	m.Data = make([]byte, len(msg.Payload)-3)

	copy(m.Data, msg.Payload[3:])

	return nil
}

//------------------------------------------------------------------------------

type PacketTransmitted struct {
	TimeOnAir_ms uint32
}

func (m *PacketTransmitted) SerializeRequest() Message {
	message := Message{
		Type:    MSG_PACKET_TRANSMITTED,
		Payload: make([]byte, 4),
	}

	binary.LittleEndian.PutUint32(message.Payload, m.TimeOnAir_ms)

	return message
}

func (m *PacketTransmitted) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_PACKET_TRANSMITTED {
		return &MessageTypeError{}
	}

	if len(msg.Payload) != 4 {
		return &MessagePayloadSizeError{}
	}

	m.TimeOnAir_ms = binary.LittleEndian.Uint32(msg.Payload)

	return nil
}

//------------------------------------------------------------------------------

type ContinuoisRSSI struct {
	RSSI_dBm int16
}

func (m *ContinuoisRSSI) SerializeRequest() Message {
	message := Message{
		Type:    MSG_CONTINUOUS_RSSI,
		Payload: make([]byte, 2),
	}

	binary.LittleEndian.PutUint16(message.Payload, uint16(m.RSSI_dBm))

	return message
}

func (m *ContinuoisRSSI) DeserializeResponse(msg *Message) error {
	if msg.Type != MSG_CONTINUOUS_RSSI {
		return &MessageTypeError{}
	}

	if len(msg.Payload) != 2 {
		return &MessagePayloadSizeError{}
	}

	m.RSSI_dBm = int16(binary.LittleEndian.Uint16(msg.Payload))

	return nil
}
