module github.com/Archie3d/waveshare-usb-lora-client

go 1.25.1

replace github.com/meshtastic/go/generated => ./gen/github.com/meshtastic/go/generated

require (
	github.com/meshtastic/go/generated v0.0.0-00010101000000-000000000000
	github.com/nats-io/nats.go v1.46.1
	github.com/stretchr/testify v1.11.1
	go.bug.st/serial v1.6.4
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/nats-io/nkeys v0.4.11 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
)

require (
	github.com/creack/goselect v0.1.3 // indirect
	golang.org/x/sys v0.36.0 // indirect
	google.golang.org/protobuf v1.36.9
)
