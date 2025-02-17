package integration

import (
	"testing"
	"time"

	"github.com/polinanime/p2pmessenger/internal/models"
	"github.com/polinanime/p2pmessenger/internal/utils"
)

func TestMessengerEndToEnd(t *testing.T) {
	// Create two messengers
	settings1 := utils.Settings{
		Username: "user1",
		Port:     "8081",
	}
	settings2 := utils.Settings{
		Username: "user2",
		Port:     "8082",
	}

	messenger1 := models.NewP2PMessenger(settings1)
	messenger2 := models.NewP2PMessenger(settings2)

	defer messenger1.Close()
	defer messenger2.Close()

	// Start both messengers
	if err := messenger1.Start(); err != nil {
		t.Fatalf("Failed to start messenger1: %v", err)
	}
	if err := messenger2.Start(); err != nil {
		t.Fatalf("Failed to start messenger2: %v", err)
	}

	// Add peers
	if err := messenger1.AddPeer(messenger2.GetAddress()); err != nil {
		t.Fatalf("Failed to add peer: %v", err)
	}

	// Send test message
	testMessage := "Hello, this is a test message"
	if err := messenger1.SendMessage("user2", testMessage); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Wait for message processing
	time.Sleep(time.Second)

	// Verify message in history
	messages := messenger2.GetHistory()
	found := false
	for _, msg := range messages {
		if msg.From == "user1" && msg.Content == testMessage {
			found = true
			break
		}
	}

	if !found {
		t.Error("Message not found in recipient's history")
	}
}
