package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Archie3d/waveshare-usb-lora-client/pkg/meshtastic"
	"github.com/charmbracelet/log"
	"github.com/nats-io/nats.go"
	"gopkg.in/yaml.v3"
)

func usage() {
	flag.PrintDefaults()
}

func showUsageAndExit(exitCode int) {
	fmt.Println("Waveshare Meshtastic Node ntfy.sh message forwarder")
	usage()
	os.Exit(exitCode)
}

type Configuration struct {
	NatsUrl           string `yaml:"nats_url"`
	NatsSubjectPrefix string `yaml:"nats_subject_prefix"`
	NtfyRecvSubject   string `yaml:"ntfy_recv_subject"`
	NtfySendSubject   string `yaml:"ntfy_send_subject"`
}

func loadConfiguration(configFile string) (*Configuration, error) {
	f, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	config := &Configuration{}
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func main() {
	var configFile = flag.String("c", "", "Configuration file")
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

	if *configFile == "" {
		log.Fatal("Configuration file is not specified")
	}

	config, err := loadConfiguration(*configFile)
	if err != nil {
		log.With("err", err).Fatal("Failed to load configuration")
	}

	nc, err := nats.Connect(config.NatsUrl)
	if err != nil {
		log.With("err", err).Fatal("Failed to connect to NATS server")
	}

	defer nc.Close()

	sub, err := nc.Subscribe(config.NatsSubjectPrefix+".in.text", func(msg *nats.Msg) {
		// Unmarshall msg.Data as JSON and extract the "text" field
		var message meshtastic.TextApplicationIncomingMessage
		err := json.Unmarshal(msg.Data, &message)
		if err != nil {
			log.With("err", err).Error("Failed to unmarshal message")
			return
		}

		log.With("from", message.From, "text", message.Text).Info("Forwarding message")

		// Forward the message to ntfy.sh
		req, err := http.NewRequest("POST", "https://ntfy.sh/"+config.NtfyRecvSubject, bytes.NewBufferString(message.Text))
		if err != nil {
			log.With("err", err).Error("Failed to create request to ntfy.sh")
			return
		}
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("Title", fmt.Sprintf("Message from node %s", message.From))
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.With("err", err).Error("Failed to forward message to ntfy.sh")
			return
		}
		defer resp.Body.Close()
	})

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		// Make sure we turn the radio off
		sub.Unsubscribe()
		os.Exit(0)
	}()

	log.Info("Ntfy service is up an running")

	select {}
}
