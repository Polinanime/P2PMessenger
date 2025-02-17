package models_test

import (
	"testing"

	"github.com/polinanime/p2pmessenger/internal/models"
	"github.com/polinanime/p2pmessenger/internal/testutils"
	"github.com/polinanime/p2pmessenger/internal/utils"
)

func TestNewDHTNode(t *testing.T) {
	settings := utils.Settings{
		Username: "testUser",
		Port:     "8080",
	}

	dht := models.NewDHTNode(settings)
	if dht == nil {
		t.Fatal("Expected non-nil DHT node")
	}
}

func TestAddNode(t *testing.T) {
	settings := utils.Settings{
		Username: "testUser",
		Port:     "8080",
	}

	dht := models.NewDHTNode(settings)

	// Create a test node with a specific address
	testAddr := "127.0.0.1:8081"
	testNode := testutils.CreateTestNodeWithAddress(t, testAddr)

	// Add the node
	if err := dht.AddPeer(testNode.Address); err != nil {
		t.Fatalf("Failed to add peer: %v", err)
	}

	// Verify the node was added
	if !dht.HasNode(testNode.Address) {
		t.Errorf("Expected node %s to be in DHT", testNode.Address)
	}
}

func TestAddNode_Multiple(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{
			name:    "Valid address",
			address: "127.0.0.1:8081",
			wantErr: false,
		},
		{
			name:    "Duplicate address",
			address: "127.0.0.1:8081",
			wantErr: true,
		},
		{
			name:    "Different address",
			address: "127.0.0.1:8082",
			wantErr: false,
		},
	}

	settings := utils.Settings{
		Username: "testUser",
		Port:     "8080",
	}
	dht := models.NewDHTNode(settings)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testNode := testutils.CreateTestNodeWithAddress(t, tt.address)
			err := dht.AddPeer(testNode.Address)

			if (err != nil) != tt.wantErr {
				t.Errorf("AddPeer() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if !dht.HasNode(testNode.Address) {
					t.Errorf("Node %s not found in DHT after addition", testNode.Address)
				}
			}
		})
	}
}
