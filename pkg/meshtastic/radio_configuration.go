package meshtastic

import (
	"fmt"
	"strconv"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/client"
	"gopkg.in/yaml.v3"
)

type LoRaBandwidth uint32

func (b LoRaBandwidth) MarshalYAML() (any, error) {
	return uint32(b), nil
}

func (b *LoRaBandwidth) UnmarshalYAML(node *yaml.Node) error {
	bw, err := strconv.ParseUint(node.Value, 10, 32)
	if err != nil {
		return err
	}

	switch bw {
	case 7:
		*b = client.LORA_BW_007
	case 10:
		*b = client.LORA_BW_010
	case 15:
		*b = client.LORA_BW_015
	case 20:
		*b = client.LORA_BW_020
	case 31:
		*b = client.LORA_BW_031
	case 41:
		*b = client.LORA_BW_041
	case 62:
		*b = client.LORA_BW_062
	case 125:
		*b = client.LORA_BW_125
	case 250:
		*b = client.LORA_BW_250
	case 500:
		*b = client.LORA_BW_500
	default:
		return fmt.Errorf("unsupported LoRa bandwidth %d", bw)
	}

	return nil
}

//------------------------------------------------------------------------------

type LoRaSpreadingFactor uint32

func (s LoRaSpreadingFactor) MarshalYAML() (any, error) {
	return uint32(s), nil
}

func (s *LoRaSpreadingFactor) UnmarshalYAML(node *yaml.Node) error {
	sf, err := strconv.ParseUint(node.Value, 10, 32)
	if err != nil {
		return err
	}

	if sf < client.LORA_SF5 || sf > client.LORA_SF12 {
		return fmt.Errorf("unsupported LoRa spresding factor %d", sf)
	}

	*s = LoRaSpreadingFactor(sf)

	return nil
}

//------------------------------------------------------------------------------

type LoRaCodingRate uint32

func (c LoRaCodingRate) MarshalYAML() (any, error) {
	cr := uint32(c)

	switch cr {
	case client.LORA_CR_4_5:
		return "4/5", nil
	case client.LORA_CR_4_6:
		return "4/6", nil
	case client.LORA_CR_4_7:
		return "4/7", nil
	case client.LORA_CR_4_8:
		return "4/8", nil
	}

	return nil, fmt.Errorf("unsupported LoRa coding rate: %d", cr)
}

func (c *LoRaCodingRate) UnmarshalYAML(node *yaml.Node) error {
	var cr uint32

	switch node.Value {
	case "4/5":
		cr = client.LORA_CR_4_5
	case "4/6":
		cr = client.LORA_CR_4_6
	case "4/7":
		cr = client.LORA_CR_4_7
	case "4/8":
		cr = client.LORA_CR_4_8
	default:
		return fmt.Errorf("unknown LoRa coding rate '%s'", node.Value)
	}

	*c = LoRaCodingRate(cr)

	return nil
}

//------------------------------------------------------------------------------

type LoRaPower int32

func (p LoRaPower) MarshalYAML() (any, error) {
	return int32(p), nil
}

func (s *LoRaPower) UnmarshalYAML(node *yaml.Node) error {
	pw, err := strconv.ParseInt(node.Value, 10, 32)
	if err != nil {
		return err
	}

	if pw != 14 && pw != 17 && pw != 20 && pw != 22 {
		return fmt.Errorf("unsupported LoRa power %d", pw)
	}

	*s = LoRaPower(pw)

	return nil
}
