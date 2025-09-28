module github.com/Archie3d/waveshare-usb-lora-client

go 1.25.1

replace github.com/meshtastic/go/generated => ./gen/github.com/meshtastic/go/generated

require (
	github.com/meshtastic/go/generated v0.0.0-00010101000000-000000000000
	go.bug.st/serial v1.6.4
)

require (
	github.com/creack/goselect v0.1.3 // indirect
	golang.org/x/sys v0.36.0 // indirect
	google.golang.org/protobuf v1.36.9
)
