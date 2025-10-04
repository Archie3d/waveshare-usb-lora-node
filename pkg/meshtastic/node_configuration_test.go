package meshtastic

import (
	"path/filepath"
	"testing"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestLoadNodeConfigYaml(t *testing.T) {
	cfg, err := LoadNodeConfiguration(filepath.Join("testdata", "node_config.yaml"))

	assert.NoError(t, err)

	assert.Equal(t, uint32(0x11223344), cfg.Id)
	assert.Equal(t, "TestNode", cfg.ShortName)
	assert.Equal(t, "TestNodeLongName", cfg.LongName)
	assert.Equal(t, types.MacAddress([]byte{0xAA, 0xBB, 0x11, 0x22, 0x33, 0x44}), cfg.MacAddress)
	assert.Equal(t, "HardwareModel", cfg.HwModel)

	assert.Equal(t, 2, len(cfg.Channels))

	assert.Equal(t, uint32(0), cfg.Channels[0].Id)
	assert.Equal(t, "LongFast", cfg.Channels[0].Name)
	assert.Equal(t, types.CryptoKey([]byte{0x01}), cfg.Channels[0].EncryptionKey)

	assert.Equal(t, uint32(1), cfg.Channels[1].Id)
	assert.Equal(t, "Private", cfg.Channels[1].Name)
}
