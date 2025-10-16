# Client for Waveshare USB-to-LoRa custom firmware

This is a client code to talk to [Waveshare USB-to-LoRa custom firmware](https://github.com/Archie3d/waveshare-usb-lora-firmware) over a serial interface.

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
