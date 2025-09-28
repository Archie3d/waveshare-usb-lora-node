package main

import (
	"fmt"
	"log"
	"time"

	pb "github.com/meshtastic/go/generated"
	"google.golang.org/protobuf/proto"

	client "github.com/Archie3d/waveshare-usb-lora-client/pkg/client"
)

const (
	//PORT = "/dev/ttyACM0"
	PORT = "COM4"
)

func main() {
	packet := &pb.MeshPacket{
		From: 0x11223344,
		To:   0x22334455,
	}

	fmt.Printf("Packet: %v\n", packet)

	serializedData, err := proto.Marshal(packet)
	if err != nil {
		log.Fatalf("Failed to marshal packet: %v", err)
	}

	fmt.Printf("Packet serialized: %v\n", serializedData)

	api := client.NewApiClient()

	err = api.Open(PORT)
	if err != nil {
		log.Fatalf("Failed to open port: %v", err)
	}

	version := api.SendRequest(&client.Version{}, time.Second)
	fmt.Printf("Version: %v\n", version)

	ena := api.SendRequest(&client.SwitchToRx{}, time.Second)
	fmt.Printf("Switch to RX %v\n", ena)

	rssi := api.SendRequest(&client.InstantaneousRSSI{}, time.Second)
	fmt.Printf("RSSI: %v\n", rssi)

	api.Close()
}
