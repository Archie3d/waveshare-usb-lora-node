//go:generate protoc --proto_path=../protobufs/ --go_out=../gen ../protobufs/meshtastic/mesh.proto
//go:generate protoc --proto_path=../protobufs/ --go_out=../gen ../protobufs/meshtastic/config.proto
//go:generate protoc --proto_path=../protobufs/ --go_out=../gen ../protobufs/meshtastic/module_config.proto
//go:generate protoc --proto_path=../protobufs/ --go_out=../gen ../protobufs/meshtastic/device_ui.proto
//go:generate protoc --proto_path=../protobufs/ --go_out=../gen ../protobufs/meshtastic/portnums.proto
//go:generate protoc --proto_path=../protobufs/ --go_out=../gen ../protobufs/meshtastic/telemetry.proto
//go:generate protoc --proto_path=../protobufs/ --go_out=../gen ../protobufs/meshtastic/channel.proto
//go:generate protoc --proto_path=../protobufs/ --go_out=../gen ../protobufs/meshtastic/xmodem.proto

package proto

// This file doesn't contain actual code; it's used to attach the `go:generate` directive
// to the Protobuf files located in this directory.
//
// Run go generate ./...
