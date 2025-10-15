package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/meshtastic"
)

func usage() {
	flag.PrintDefaults()
}

func showUsageAndExit(exitCode int) {
	fmt.Println("Waveshare USB LoRa Meshtastic Node")
	usage()
	os.Exit(exitCode)
}

func main() {
	var configFile = flag.String("c", "", "Configuration file")
	var serialPort = flag.String("p", "", "Serial port")
	var showHelp = flag.Bool("h", false, "Show help")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if *showHelp {
		showUsageAndExit(0)
	}

	if *serialPort == "" {
		log.Fatal("Serial port is not specified")
	}

	if *configFile == "" {
		log.Fatal("Configuration file is not specified")
	}

	config, err := meshtastic.LoadNodeConfiguration(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %s", err.Error())
	}

	node := meshtastic.NewNode(*serialPort, config)

	node.AddApplication(meshtastic.NewTextApplication())

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

	//log.Println("Waiting for the device to initialize")
	//<-time.After(time.Second)

	/*
		log.Println("Sending message")

		err = node.SendText(0, 0xFFFFFFFF, "Hello from Waveshare USB LoRa!")
		if err != nil {
			log.Printf("SendText failed: %v\n", err)
		}
	*/

	select {}
}
