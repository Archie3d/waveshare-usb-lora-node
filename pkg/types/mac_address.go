package types

import (
	"encoding/hex"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type MacAddress [6]byte

func (m MacAddress) AsByteArray() []byte {
	return m[0:6]
}

func (m MacAddress) MarshalYAML() (any, error) {
	var value string = ""
	for _, b := range m {
		if len(value) > 0 {
			value = value + ":"
		}
		value = value + fmt.Sprintf("%x", b)
	}

	return value, nil
}

func (m *MacAddress) UnmarshalYAML(node *yaml.Node) error {
	chunks := strings.Split(node.Value, ":")
	mac := make([]byte, 0)

	for _, c := range chunks {
		b, err := hex.DecodeString(c)
		if err != nil {
			return err
		}
		mac = append(mac, b...)
	}

	if len(mac) != 6 {
		return fmt.Errorf("mac address length is invalid")
	}

	*m = MacAddress(mac)

	return nil
}
