// Publisher example — creates an IPC channel and sends messages.
package main

import (
	"fmt"
	"log"
	"time"

	zigbolt "github.com/suenot/zigbolt-go"
)

func main() {
	major, minor, patch := zigbolt.Version()
	fmt.Printf("ZigBolt version: %d.%d.%d\n", major, minor, patch)

	const channelName = "/zigbolt-go-example"

	ch, err := zigbolt.CreateChannel(channelName, zigbolt.DefaultTermLength)
	if err != nil {
		log.Fatalf("CreateChannel: %v", err)
	}
	defer ch.Close()

	fmt.Printf("Publishing on channel %s ...\n", channelName)

	for i := 0; i < 10; i++ {
		msg := fmt.Sprintf("Hello from Go publisher #%d", i)
		if err := ch.Publish([]byte(msg), 1); err != nil {
			log.Printf("Publish error: %v", err)
		} else {
			fmt.Printf("Sent: %s\n", msg)
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("Done.")
}
