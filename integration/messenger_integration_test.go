package integration

import (
	"log"
	"testing"
	"time"

	"github.com/polinanime/p2pmessenger/internal/testutils"
)

func TestMessengerEndToEnd(t *testing.T) {
	messenger1, messenger2 := testutils.CreateMessengers(t)
	defer messenger1.Close()
	defer messenger2.Close()

	log.Printf("Eve's address: %s", messenger1.GetAddress())
	log.Printf("Adam's address: %s", messenger2.GetAddress())

	// Verify peers are connected
	if !messenger1.GetDHT().HasNode(messenger2.GetAddress()) {
		t.Fatal("Messenger1 does not have Messenger2 in its DHT")
	}
	if !messenger2.GetDHT().HasNode(messenger1.GetAddress()) {
		t.Fatal("Messenger2 does not have Messenger1 in its DHT")
	}

	// Send test message
	testMessage := "Hello, this is a test message"
	if err := messenger1.SendMessage("Adam", testMessage); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Wait for message processing
	time.Sleep(time.Second)

	// Verify message in history
	messages := messenger2.GetHistory()
	found := false
	for _, msg := range messages {
		log.Printf("Message in history: From=%s, To=%s, Content=%s",
			msg.From, msg.To, msg.Content)
		if msg.From == "Eve" && msg.Content == testMessage {
			found = true
			break
		}
	}

	if !found {
		t.Error("Message not found in recipient's history")
	}
}
