package types

import (
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

type NodeId uint32

func (n NodeId) MarshalYAML() (any, error) {
	return n.String(), nil
}

func (n *NodeId) UnmarshalYAML(node *yaml.Node) error {
	value, err := strconv.ParseUint(node.Value, 16, 32)
	if err != nil {
		return err
	}

	*n = NodeId(value)

	return nil
}

func (n NodeId) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%08x\"", uint32(n))), nil
}

func (n *NodeId) UnmarshalJSON(data []byte) error {
	value := string(data)
	if len(value) > 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}

	num, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return err
	}

	*n = NodeId(num)
	return nil
}

func (n NodeId) String() string {
	return fmt.Sprintf("%x", uint32(n))
}
