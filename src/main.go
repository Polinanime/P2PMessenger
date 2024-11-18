package main

import (
	"fmt"
	"log"
	"time"

	. "github.com/polinanime/p2pmessenger/models"
)

func main() {
	fmt.Println("Starting P2P Messenger")

	// Create messengers
	messenger1 := NewP2PMessenger("Adam", "1235")
	messenger2 := NewP2PMessenger("Eve", "1236")
	messenger3 := NewP2PMessenger("Bob", "1237")

	defer messenger1.Close()
	defer messenger2.Close()
	defer messenger3.Close()

	// Channel to wait for all messengers to complete bootstrap
	done := make(chan bool)
	messengers := []*P2PMessenger{messenger1, messenger2, messenger3}

	// First, start all listeners
	for i, m := range messengers {
		go func(index int, messenger *P2PMessenger) {
			if err := messenger.Start(); err != nil {
				log.Printf("Error starting messenger%d: %v", index+1, err)
			}
			done <- true
		}(i, m)
	}

	// Wait for all listeners to be ready
	log.Println("Waiting for listeners to be ready...")
	for _, m := range messengers {
		<-m.Ready
	}

	// Wait for bootstrap to complete
	log.Println("Waiting for bootstrap to complete...")
	for range messengers {
		<-done
	}

	time.Sleep(time.Second) // Give time for final setup

	// Print DHT states
	for i, m := range messengers {
		fmt.Printf("\nMessenger%d DHT:\n", i+1)
		m.PrintDht()
	}

	// Send test message
	log.Println("\nSending test message...")
	err := messenger1.SendMessage("Eve", "Hello, Eve!")
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}

	// Keep the program running
	select {}
}
