package testutils

import (
	"crypto/sha3"
	"testing"
	"time"

	"github.com/polinanime/p2pmessenger/internal/models"
	"github.com/polinanime/p2pmessenger/internal/types"
	"github.com/polinanime/p2pmessenger/internal/utils"
)

// CreateTestNodeWithAddress creates a node with a specific address for testing
func CreateTestNodeWithAddress(t *testing.T, address string) types.Node {
	t.Helper()

	// Create ID from address
	id := sha3.Sum256([]byte(address))

	return types.Node{
		ID:      id[:],
		Address: address,
	}
}

// CreateTestNode creates a node with default address for testing
func CreateTestNode(t *testing.T) types.Node {
	return CreateTestNodeWithAddress(t, "127.0.0.1:8081")
}

// CreateTestNetwork creates a network of nodes for testing
func CreateTestNetwork(t *testing.T, size int) []types.Node {
	t.Helper()

	nodes := make([]types.Node, size)
	for i := 0; i < size; i++ {
		nodes[i] = CreateTestNode(t)
	}
	return nodes
}

// SetupMessengers creates two messengers for testing
func CreateMessengers(t *testing.T) (*models.P2PMessenger, *models.P2PMessenger) {
	settings1 := utils.Settings{
		Username: "Eve",
		Port:     "8081",
	}
	settings2 := utils.Settings{
		Username: "Adam",
		Port:     "8082",
	}

	messenger1 := models.NewP2PMessenger(settings1)
	messenger2 := models.NewP2PMessenger(settings2)

	// Start both messengers
	if err := messenger1.Start(); err != nil {
		t.Fatalf("Failed to start messenger1: %v", err)
	}
	if err := messenger2.Start(); err != nil {
		t.Fatalf("Failed to start messenger2: %v", err)
	}

	// Wait for startup
	time.Sleep(time.Second)

	// Connect peers
	if err := messenger1.AddPeer(messenger2.GetAddress()); err != nil {
		t.Fatalf("Failed to add peer1: %v", err)
	}
	if err := messenger2.AddPeer(messenger1.GetAddress()); err != nil {
		t.Fatalf("Failed to add peer2: %v", err)
	}

	// Wait for connections
	time.Sleep(time.Second)

	return messenger1, messenger2
}
