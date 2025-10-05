package types

import (
	"time"

	"gopkg.in/yaml.v3"
)

type Duration time.Duration

func (k Duration) MarshalYAML() (any, error) {
	return time.Duration(k).String(), nil
}

func (k *Duration) UnmarshalYAML(node *yaml.Node) error {
	tmp, err := time.ParseDuration(node.Value)
	if err != nil {
		return err
	}
	*k = Duration(tmp)
	return nil
}
