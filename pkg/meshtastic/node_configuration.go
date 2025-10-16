package meshtastic

import (
	"os"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/types"
	"gopkg.in/yaml.v3"
)

// See https://dev.to/ilyakaznacheev/a-clean-way-to-pass-configs-in-a-go-application-1g64

type NodeConfiguration struct {
	Id         uint32           `yaml:"id"`
	ShortName  string           `yaml:"short_name"`
	LongName   string           `yaml:"long_name"`
	MacAddress types.MacAddress `yaml:"mac_address"`
	HwModel    uint32           `yaml:"hw_model"`
	PublicKey  types.CryptoKey  `yaml:"public_key"`

	NatsUrl           string `yaml:"nats_url"`
	NatsSubjectPrefix string `yaml:"nats_subject_prefix"`

	Radio RadioConfiguration `yaml:"radio"`

	Channels []ChannelConfiguration `yaml:"channels"`
}

type RadioConfiguration struct {
	Frequency       uint32              `yaml:"frequency"`
	Power           LoRaPower           `yaml:"power"`
	SpreadingFactor LoRaSpreadingFactor `yaml:"spreading_factor"`
	Bandwidth       LoRaBandwidth       `yaml:"bandwidth"`
	CodingRate      LoRaCodingRate      `yaml:"coding_rate"`
}

type ChannelConfiguration struct {
	Id            uint32          `yaml:"id"`
	Name          string          `yaml:"name"`
	EncryptionKey types.CryptoKey `yaml:"encryption_key"`
}

func LoadNodeConfiguration(configFile string) (*NodeConfiguration, error) {
	f, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	config := &NodeConfiguration{}
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
