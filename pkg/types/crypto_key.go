package types

import (
	"encoding/base64"

	"gopkg.in/yaml.v3"
)

type CryptoKey []byte

func (k CryptoKey) MarshalYAML() (any, error) {
	return k.String(), nil
}

func (k *CryptoKey) UnmarshalYAML(node *yaml.Node) error {
	value := node.Value

	ba, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return err
	}
	*k = ba
	return nil
}

func (k CryptoKey) String() string {
	return base64.StdEncoding.EncodeToString(k)
}
