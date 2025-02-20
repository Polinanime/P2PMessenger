package models_test

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/polinanime/p2pmessenger/internal/models"
	"github.com/polinanime/p2pmessenger/internal/utils"
)

func TestSendMessage(t *testing.T) {
	// Generate a test key pair for encryption
	keyPair, err := models.GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Export public key
	publicKeyPEM, err := keyPair.ExportPublicKey()
	if err != nil {
		t.Fatalf("Failed to export public key: %v", err)
	}

	tests := []struct {
		name        string
		recipient   string
		message     string
		expectError bool
	}{
		{
			name:        "successful message send",
			recipient:   "Bob",
			message:     "Hello, Bob!",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock connection
			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()

			// Channel to receive server data
			receivedData := make(chan []byte, 1)
			done := make(chan bool)

			// Start a goroutine to read from the server side
			go func() {
				defer close(receivedData)
				buf := make([]byte, 4096)
				n, err := serverConn.Read(buf)
				if err != nil {
					t.Logf("Server read error: %v", err)
					return
				}
				receivedData <- buf[:n]
				done <- true
			}()

			settings := utils.Settings{
				Username: "Alice",
				Port:     "8080",
			}

			// Create messenger with mock
			messenger := models.NewMockMessenger(settings, clientConn)

			// Mock DHT data for the recipient
			recipientInfo := map[string]string{
				"userID":    tt.recipient,
				"address":   "localhost:8081",
				"publicKey": string(publicKeyPEM),
			}
			infoBytes, _ := json.Marshal(recipientInfo)

			// Add mock data to DHT
			messenger.GetDHT().Store(tt.recipient, string(infoBytes))

			// Try to send message
			err := messenger.SendMessage(tt.recipient, tt.message)

			// Check error expectation
			if (err != nil) != tt.expectError {
				t.Errorf("SendMessage() error = %v, expectError %v", err, tt.expectError)
			}

			// For successful cases, verify the message was encrypted and sent
			if !tt.expectError {
				// Wait for data with timeout
				select {
				case data := <-receivedData:
					var sentMsg models.Message
					if err := json.Unmarshal(data, &sentMsg); err != nil {
						t.Fatalf("Failed to unmarshal written data: %v", err)
					}

					// Decrypt the message to verify content
					decryptedContent, err := keyPair.Decrypt(sentMsg.EncryptedContent)
					if err != nil {
						t.Fatalf("Failed to decrypt message: %v", err)
					}

					if string(decryptedContent) != tt.message {
						t.Errorf("Wrong message content. Got %s, want %s",
							string(decryptedContent), tt.message)
					}
				case <-time.After(2 * time.Second):
					t.Fatal("Test timed out waiting for message")
				}
			}
		})
	}
}
