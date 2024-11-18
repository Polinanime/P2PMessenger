package models

import (
	"bufio"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Node struct {
	ID      []byte
	Address string
}

type DHTNode struct {
	node         Node
	routingTable map[string]Node
	dataStore    map[string]string
	mutex        sync.RWMutex
}

type DHTRequest struct {
	Type    string
	Payload interface{}
}

type DHTResponse struct {
	RoutingTable map[string]Node
	DataStore    map[string]string
}

func NewDHTNode(address string) *DHTNode {
	id := sha1.Sum([]byte(address))
	dht := &DHTNode{
		node: Node{
			ID:      id[:],
			Address: address,
		},
		routingTable: make(map[string]Node),
		dataStore:    make(map[string]string),
	}

	err := dht.loadPeersFromConfig()

	if err != nil {
		log.Println("Warning: failed to load peers from configuration file")
	}
	return dht
}

func (dht *DHTNode) Bootstrap() error {
	log.Printf("[%s] Starting bootstrap", dht.node.Address)

	dht.mutex.RLock()
	peers := make([]Node, 0, len(dht.routingTable))
	for _, node := range dht.routingTable {
		peers = append(peers, node)
	}
	dht.mutex.RUnlock()

	maxRetries := 3
	retryDelay := time.Second

	for _, node := range peers {
		if node.Address == dht.node.Address {
			continue
		}

		var connected bool
		for retry := 1; retry <= maxRetries; retry++ {
			log.Printf("[%s] Attempting to connect to %s (try %d/%d)",
				dht.node.Address, node.Address, retry, maxRetries)

			if err := dht.connectToPeer(node); err != nil {
				log.Printf("[%s] Failed to connect to %s (attempt %d): %v",
					dht.node.Address, node.Address, retry, err)
				time.Sleep(retryDelay)
				continue
			}

			connected = true
			log.Printf("[%s] Successfully connected to %s", dht.node.Address, node.Address)
			break
		}

		if !connected {
			log.Printf("[%s] Failed to connect to %s after %d attempts",
				dht.node.Address, node.Address, maxRetries)
		}
	}

	return nil
}

func (dht *DHTNode) connectToPeer(node Node) error {
	conn, err := net.DialTimeout("tcp", node.Address, time.Second*3)
	if err != nil {
		return fmt.Errorf("dial failed: %v", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(time.Second * 5))

	request := DHTRequest{
		Type: "GET_DHT",
	}

	log.Printf("[%s] Sending DHT request to %s", dht.node.Address, node.Address)

	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	var response DHTResponse
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return fmt.Errorf("failed to receive response: %v", err)
	}

	log.Printf("[%s] Received response from %s", dht.node.Address, node.Address)

	// Update local DHT with unique entries
	dht.mutex.Lock()
	for _, node := range response.RoutingTable {
		// Use node ID as key to ensure uniqueness
		dht.routingTable[string(node.ID)] = node
	}
	for k, v := range response.DataStore {
		dht.dataStore[k] = v
	}
	dht.mutex.Unlock()

	return nil
}

func (dht *DHTNode) addNode(node Node) {
	dht.mutex.Lock()
	defer dht.mutex.Unlock()
	dht.routingTable[string(node.ID)] = node
}

func (dht *DHTNode) hasNode(address string) bool {
	dht.mutex.RLock()
	defer dht.mutex.RUnlock()

	for _, node := range dht.routingTable {
		if node.Address == address {
			return true
		}
	}
	return false
}

func (dht *DHTNode) HandleDHTRequest(conn net.Conn) {
	defer conn.Close()

	var request DHTRequest
	if err := json.NewDecoder(conn).Decode(&request); err != nil {
		log.Printf("[%s] Error decoding DHT request: %v", dht.node.Address, err)
		return
	}

	log.Printf("[%s] Received DHT request type: %s", dht.node.Address, request.Type)

	switch request.Type {
	case "GET_DHT":
		dht.mutex.RLock()
		response := DHTResponse{
			RoutingTable: dht.routingTable,
			DataStore:    dht.dataStore,
		}
		dht.mutex.RUnlock()

		if err := json.NewEncoder(conn).Encode(response); err != nil {
			log.Printf("[%s] Error sending DHT response: %v", dht.node.Address, err)
			return
		}

		log.Printf("[%s] DHT response sent", dht.node.Address)
	}
}

func (dht *DHTNode) loadPeersFromConfig() error {
	configPath := ".config/peers.txt"

	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Use a map to track unique addresses
	uniquePeers := make(map[string]Node)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Create node ID using address
		id := sha1.Sum([]byte(line))
		node := Node{
			ID:      id[:],
			Address: line,
		}

		// Store using address as key to ensure uniqueness
		uniquePeers[line] = node
	}

	// Now add unique peers to routing table
	dht.mutex.Lock()
	for _, node := range uniquePeers {
		dht.routingTable[string(node.ID)] = node
	}
	dht.mutex.Unlock()

	return scanner.Err()
}

func (dht *DHTNode) savePeersToConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Ensure .config directory exists
	configDir := filepath.Join(homeDir, ".config")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return err
	}

	// Open file for writing
	configPath := filepath.Join(configDir, "peers.txt")
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write each peer's address to file
	dht.mutex.RLock()
	defer dht.mutex.RUnlock()

	for _, node := range dht.routingTable {
		_, err := file.WriteString(node.Address + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func (dht *DHTNode) Store(key, value string) error {
	dht.mutex.Lock()
	defer dht.mutex.Unlock()
	dht.dataStore[key] = value
	return nil
}

func (dht *DHTNode) Get(key string) (string, bool) {
	dht.mutex.RLock()
	defer dht.mutex.RUnlock()
	value, ok := dht.dataStore[key]
	return value, ok
}

func (dht *DHTNode) FindNode(target []byte) Node {
	dht.mutex.RLock()
	defer dht.mutex.RUnlock()
	return dht.routingTable[string(target)]
}

func (dht *DHTNode) RegisterUser(userID string, address string) error {
	dht.mutex.Lock()
	defer dht.mutex.Unlock()

	// Create user info
	userInfo := map[string]string{
		"userID":   userID,
		"address":  address,
		"status":   "online",
		"lastSeen": time.Now().String(),
	}

	// Convert to JSON
	infoBytes, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}

	// Store in dataStore
	dht.dataStore[userID] = string(infoBytes)
	return nil
}

func (dht *DHTNode) LookupUser(userID string) (map[string]string, error) {
	dht.mutex.RLock()
	defer dht.mutex.RUnlock()

	data, exists := dht.dataStore[userID]
	if !exists {
		return nil, fmt.Errorf("user %s not found", userID)
	}

	var userInfo map[string]string
	err := json.Unmarshal([]byte(data), &userInfo)
	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

func (dht *DHTNode) Print() {
	fmt.Println("\n=== Routing Table ===")
	fmt.Println("Address -> NodeID")

	// Create a map to check for duplicates
	addresses := make(map[string][]string)

	for _, node := range dht.routingTable {
		nodeIDHex := fmt.Sprintf("%x", node.ID)
		addresses[node.Address] = append(addresses[node.Address], nodeIDHex)
	}

	// Print unique addresses and their IDs
	for addr, ids := range addresses {
		fmt.Printf("%s -> %v\n", addr, ids)
	}

	fmt.Println("\n=== Data Store ===")
	fmt.Println("UserID -> UserInfo")
	for userID, data := range dht.dataStore {
		var userInfo map[string]string
		json.Unmarshal([]byte(data), &userInfo)
		fmt.Printf("%s -> %v\n", userID, userInfo)
	}
}
