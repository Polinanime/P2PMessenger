package main_test

import (
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.Println("Setting up tests...")

	if err := setup(); err != nil {
		log.Printf("Test setup failed: %v", err)
		os.Exit(1)
	}

	code := m.Run()

	log.Println("Cleaning up after tests...")
	if err := teardown(); err != nil {
		log.Printf("Test cleanup failed: %v", err)
	}

	os.Exit(code)
}

func setup() error {
	// Create test config directory
	if err := os.MkdirAll(".config", 0755); err != nil {
		return err
	}
	return nil
}

func teardown() error {
	// Clean up test config directory
	return os.RemoveAll(".config")
}
