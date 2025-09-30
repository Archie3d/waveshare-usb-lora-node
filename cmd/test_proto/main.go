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

	node := meshtastic.NewNode()
	if err := node.Start(); err != nil {
		log.Fatal(err)
	}

	defer node.Stop()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		// Make sure we turn the radio off
		node.Stop()
		os.Exit(0)
	}()

	select {}
}
