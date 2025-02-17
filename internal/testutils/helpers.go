package testutils

import (
	"crypto/sha3"
	"net"
	"testing"

	"github.com/polinanime/p2pmessenger/internal/types"
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

// MockConnection creates a mock network connection for testing
func MockConnection() (net.Conn, net.Conn) {
	conn1, conn2 := net.Pipe()
	return conn1, conn2
}
