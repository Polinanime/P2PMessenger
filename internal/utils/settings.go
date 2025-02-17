package utils

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/polinanime/p2pmessenger/internal/types"
	"golang.org/x/crypto/sha3"
)

type Settings struct {
	Username   string
	Address    string
	Port       string
	configPath string
}

func NewSettings(username, port, configPath string) *Settings {
	return &Settings{
		Username:   username,
		Port:       port,
		configPath: configPath,
	}
}

func (s *Settings) Save() error {
	return nil
}

func (s *Settings) Load() error {
	return nil
}

func (s *Settings) ReadPeers(ignoreAddress string) (map[string]types.Node, error) {
	configPath := ".config/peers.txt"
	log.Printf("Loading peers from %s", configPath)

	file, err := os.Open(configPath)
	if err != nil {

		// Create config directory if it doesn't exist
		err = os.MkdirAll(".config", 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create config directory: %v", err)
		}

		// Create empty config file if it doesn't exist
		file, err = os.Create(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create config file: %v", err)
		}
		defer file.Close()

		return nil, nil
	}

	defer file.Close()

	uniquePeers := make(map[string]types.Node)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		log.Printf("Processing peer: %s", line)

		// Skip if this is our own address
		if line == ignoreAddress {
			log.Printf("Skipping ignore address")
			continue
		}

		// Create node ID using address
		id := sha3.Sum256([]byte(line))
		node := types.Node{
			ID:      id[:],
			Address: line,
		}

		log.Printf("Created node with ID %x for address %s",
			node.ID, node.Address)

		uniquePeers[line] = node
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	return uniquePeers, nil
}

func (s *Settings) CreateNode() types.Node {
	id := sha3.Sum256([]byte(s.Address))
	return types.Node{
		ID:      id[:],
		Address: s.Address,
	}
}
