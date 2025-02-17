package models

import (
	"bytes"
	"math/bits"

	"github.com/polinanime/p2pmessenger/internal/types"
)

type KBucket struct {
	Nodes []types.Node
}

// Distance returns the XOR distance between two byte slices.
func Distance(a, b []byte) []byte {
	if len(a) != len(b) {
		// Handle error case, return max distance
		return bytes.Repeat([]byte{255}, len(a))
	}

	distance := make([]byte, len(a))
	for i := 0; i < len(a); i++ {
		distance[i] = a[i] ^ b[i]
	}
	return distance
}

// CommonPrefixLength returns the length of the common prefix of two byte slices.
func CommonPrefixLength(a, b []byte) int {
	// Check length first
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	// Then compare byte by byte
	for i := 0; i < minLen; i++ {
		xor := a[i] ^ b[i]
		if xor != 0 {
			return i*8 + bits.LeadingZeros8(xor)
		}
	}
	return minLen * 8
}
