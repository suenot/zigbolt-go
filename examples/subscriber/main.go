// Subscriber example — opens an existing IPC channel and polls for messages.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	zigbolt "github.com/suenot/zigbolt-go"
)

func main() {
	major, minor, patch := zigbolt.Version()
	fmt.Printf("ZigBolt version: %d.%d.%d\n", major, minor, patch)

	const channelName = "/zigbolt-go-example"

	ch, err := zigbolt.OpenChannel(channelName, zigbolt.DefaultTermLength)
	if err != nil {
		log.Fatalf("OpenChannel: %v", err)
	}
	defer ch.Close()

	fmt.Printf("Subscribing on channel %s ... (Ctrl+C to stop)\n", channelName)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	handler := func(data []byte, msgTypeId int32) {
		fmt.Printf("Received [type=%d]: %s\n", msgTypeId, string(data))
	}

	for {
		select {
		case <-sigCh:
			fmt.Println("\nShutting down.")
			return
		default:
			ch.Poll(handler, 10)
		}
	}
}
