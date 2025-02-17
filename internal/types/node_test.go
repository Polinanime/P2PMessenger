package types_test

import (
	"testing"

	"github.com/polinanime/p2pmessenger/internal/types"
)

func TestNode(t *testing.T) {
	id := []byte("testID")
	address := "localhost:8080"

	node := types.Node{
		ID:      id,
		Address: address,
	}

	if string(node.ID) != "testID" {
		t.Errorf("Expected ID testID, got %s", string(node.ID))
	}
	if node.Address != address {
		t.Errorf("Expected address %s, got %s", address, node.Address)
	}
}
