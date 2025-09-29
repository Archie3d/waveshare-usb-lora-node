package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/meshtastic"
)

const (
	//PORT = "/dev/ttyACM0"
	PORT = "COM4"
)

func main() {
	mesh := meshtastic.NewMeshtasticClient()

	err := mesh.Open(PORT)
	if err != nil {
		log.Fatal(err)
	}

	defer mesh.Close()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		// Make sure we turn the radio off
		mesh.Close()
		os.Exit(0)
	}()

	select {}
}
