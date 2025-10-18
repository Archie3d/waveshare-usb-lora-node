package types

import (
	"time"

	"gopkg.in/yaml.v3"
)

type Duration time.Duration

func (d Duration) MarshalYAML() (any, error) {
	return d.String(), nil
}

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	tmp, err := time.ParseDuration(node.Value)
	if err != nil {
		return err
	}
	*d = Duration(tmp)
	return nil
}

func (d Duration) String() string {
	return time.Duration(d).String()
}
