package utils_test

import (
	"testing"

	"github.com/polinanime/p2pmessenger/internal/utils"
)

func TestNewSettings(t *testing.T) {
	username := "testUser"
	port := "8080"
	configPath := ".config/peers.txt"

	settings := utils.NewSettings(username, port, configPath)

	if settings.Username != username {
		t.Errorf("Expected username %s, got %s", username, settings.Username)
	}
	if settings.Port != port {
		t.Errorf("Expected port %s, got %s", port, settings.Port)
	}
}
