package main

import (
	"fmt"
	"log"
	"os"

	. "github.com/polinanime/p2pmessenger/models"
)

func main() {
	// Get environment variables
	userID := os.Getenv("USER_ID")
	port := os.Getenv("PORT")

	if userID == "" || port == "" {
		log.Fatal("USER_ID and PORT environment variables are required")
	}

	// Create messenger
	messenger := NewP2PMessenger(userID, port)
	defer messenger.Close()

	// Start the messenger
	if err := messenger.Start(); err != nil {
		log.Fatalf("Error starting messenger: %v", err)
	}

	// Print DHT state
	fmt.Printf("\nMessenger DHT for %s:\n", userID)
	messenger.PrintDht()

	// Keep the program running
	select {}
}
