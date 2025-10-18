package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/meshtastic"
	"github.com/charmbracelet/log"
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
	var logLevel = flag.String("l", "info", "Log level")
	var showHelp = flag.Bool("h", false, "Show help")

	flag.Usage = usage
	flag.Parse()

	if *showHelp {
		showUsageAndExit(0)
	}

	switch *logLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.Fatalf("Invalid log level '%s'", *logLevel)
	}

	if *serialPort == "" {
		log.Fatal("Serial port is not specified")
	}

	if *configFile == "" {
		log.Fatal("Configuration file is not specified")
	}

	config, err := meshtastic.LoadNodeConfiguration(*configFile)
	if err != nil {
		log.With("err", err).Fatal("Failed to load configuration")
	}

	node := meshtastic.NewNode(*serialPort, config)

	node.AddApplication(meshtastic.NewTextApplication(config))

	if config.NodeInfo != nil {
		node.AddApplication(meshtastic.NewNodeInfoApplication(config))
	}

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

	log.Info("Node is up an running")

	select {}
}
