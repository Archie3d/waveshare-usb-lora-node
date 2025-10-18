# Meshtastic node for Waveshare USB-to-LoRa

This node is implemented on top of [Waveshare USB-to-LoRa with custom firmware](https://github.com/Archie3d/waveshare-usb-lora-firmware). It talks to the device over a serial interface. Basic Meshtastic logic is implemented.

The node connects to a [NATS server](https://github.com/nats-io/nats-server) to receive and send application specific messages. All NATS messages are serialized to JSON.

## Generating protobufs

Install Protocol Buffers compiler [as described here](https://protobuf.dev/installation/).

```shell
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```
Make sure `$HOME/go/bin` is in the `$PATH`.

## Compiling
Using [Taskfile](https://taskfile.dev/)
```
task build
```

## Using serial port
On Linux the serial device may appear like `/dev/ttyACM0`.
To allow a non-root access:
```bash
sudo usermod -aG dialout $USER
```

## Node configuration
Node configuration should be provided as YAML file. Here is an example configuration:

```yaml
id: 0x12345678      # Node unique ID
short_name: "Nd"    # Node short name (keep it to 2-4 characters)
long_name: "My Node Name"   # Node's long name
mac_address: "AB:CD:12:34:56:78"    # Node MAC address (derived from node ID)
                                    # Deprecated, but is used for interoperability
                                    # with other Meshtastic devices
hw_model: 255       # Hardware model number, 255 corresponds to a private hardware.

public_key: "NCTMT7FWcJdyuAeJMaNMzImjRDv6nDovf/W5qIaGe/w="  # AES256 key as base64

nats_url: "nats://localhost:4222"   # URL to the NATS server
nats_subject_prefix: "mesh.my_node" # NATS messages perfix (will used at the start of all
                                    # subject names)

radio:
  frequency: 869525000      # Frequency in Hz. This value here is for the public Meshtastic
  power: 17                 # Transmission power in dBm. Allowed values: 14, 17, 20, 22
  spreading_factor: 11      # LoRa parameters
  bandwidth: 250            # Keep these values for Meshtastic LongFast communication
  coding_rate: "4/5"

retransmit:
  forward: true             # Whether to forward received packets (public or unknown)
  period: [ "3s", "7s"]     # Outgoing packets retransmission periods
  jitter: "1s"              # Random delay (from to this value) will added to the retransmission period

channels:
  - id: 0                   # Channel ID
    name: "LongFast"        # Channel name, keep "LongFast" for Meshtastic
    encryption_key: "AQ=="  # Encryption key,this one is for public Meshtastic channel 0
  - id: 1
    name: "Private"
    encryption_key: "NRHtkaJFJyV1ftZ6GluFNR1rBr3MeqHvBmyIKaho4VY="

node_info:              # Parameters used by the Node Info app
  channel: 0            # Transmission channel number (usually 0)
  publish_period: "3h"  # Broadcast period (how often this node info will be sent out)
```
