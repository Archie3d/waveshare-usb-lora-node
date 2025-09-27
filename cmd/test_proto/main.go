package main

import (
	"fmt"
	"log"

	pb "github.com/meshtastic/go/generated"
	"google.golang.org/protobuf/proto"

	client "github.com/Archie3d/waveshare-usb-lora-client/pkg/client"
)

const (
	PORT = "/dev/ttyACM0"
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

	client := client.NewApiClient()

	err = client.Open(PORT)
	if err != nil {
		log.Fatalf("Failed to open port: %v", err)
	}

	version, err := client.GetVersion()
	if err != nil {
		log.Fatalf("Failed to get version: %v", err)
	}

	fmt.Printf("Version: %v\n", version)

	rssi, err := client.GetRSSI()

	if err != nil {
		log.Fatalf("Failed to get rssi: %v", err)
	}

	fmt.Printf("RSSI: %v\n", rssi)

	err = client.SetRx(10000, true)
	if err != nil {
		log.Fatalf("Failed to set rx: %v", err)
	}

	err = client.Wait(15000)
	if err != nil {
		log.Fatalf("Failed to wait: %v", err)
	}

	err = client.Close()
	if err != nil {
		log.Fatalf("Failed to close port: %v", err)
	}

	/*
			err = ioutil.WriteFile("packet.bin", serializedData, 0644)
		    if err != nil {
		        log.Fatalf("Failed to write to file: %v", err)
		    }

		    fmt.Println("Packet serialized successfully!")
	*/
}
