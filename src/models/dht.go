package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	. "github.com/polinanime/p2pmessenger/types"
	"github.com/polinanime/p2pmessenger/utils"
	"golang.org/x/crypto/sha3"
)

// DHTNode represents a node in the DHT
type DHTNode struct {
	node      Node
	kBuckets  [KEY_SIZE]*KBucket
	dataStore map[string]string
	mutex     sync.RWMutex
}

// DHTRequest is a request to the DHT
type DHTRequest struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// DHTResponse is the response to a DHT request
type DHTResponse struct {
	Nodes     []Node
	DataStore map[string]string
}

// NewDHTNode creates a new DHT node with the given address
func NewDHTNode(settings utils.Settings) *DHTNode {
	node := settings.CreateNode()

	dht := &DHTNode{
		node:      node,
		dataStore: make(map[string]string),
	}

	// Initialize k-buckets
	for i := 0; i < KEY_SIZE; i++ {
		dht.kBuckets[i] = &KBucket{
			Nodes: make([]Node, 0, K_BUCKET_SIZE),
		}
	}

	peers, err := settings.ReadPeers(node.Address)
	if err != nil {
		log.Println("Warning: failed to load peers from configuration file")
	} else {
		dht.loadPeersFromConfig(peers)
	}

	return dht
}

// getBucketIndex determines which k-bucket a node belongs in
func (dht *DHTNode) getBucketIndex(nodeId []byte) int {
	return CommonPrefixLength(dht.node.ID, nodeId)
}

// addToKBucket adds a node to the appropriate k-bucket
func (dht *DHTNode) addToKBucket(node Node) {
	if bytes.Equal(node.ID, dht.node.ID) {
		return
	}

	dht.mutex.Lock()
	defer dht.mutex.Unlock()

	bucketIndex := dht.getBucketIndex(node.ID)
	if bucketIndex >= KEY_SIZE {
		return
	}
	if dht.kBuckets[bucketIndex] == nil {
		dht.kBuckets[bucketIndex] = &KBucket{
			Nodes: make([]Node, 0, K_BUCKET_SIZE),
		}
	}
	bucket := dht.kBuckets[bucketIndex]

	// Check if node already exists
	for i, n := range bucket.Nodes {
		if bytes.Equal(n.ID, node.ID) {
			// Move to end (most recently seen)
			bucket.Nodes = append(bucket.Nodes[:i], bucket.Nodes[i+1:]...)
			bucket.Nodes = append(bucket.Nodes, node)
			return
		}
	}

	// Add new node if bucket isn't full
	if len(bucket.Nodes) < K_BUCKET_SIZE {
		bucket.Nodes = append(bucket.Nodes, node)
		return
	}
	// Easier to discard but better to ping
	log.Printf("[%s] K-bucket %d is full, discarding new node %s",
		dht.node.Address, bucketIndex, node.Address)
}

// findClosestNodes finds the k closest nodes to a target ID
func (dht *DHTNode) findClosestNodes(targetID []byte, k int) []Node {
	dht.mutex.RLock()
	defer dht.mutex.RUnlock()

	// Create a map to store distances
	distances := make(map[string][]byte)
	var nodes []Node

	// Calculate distances for all nodes in k-buckets
	for _, bucket := range dht.kBuckets {
		for _, node := range bucket.Nodes {
			dist := Distance(targetID, node.ID)
			distances[string(node.ID)] = dist
			nodes = append(nodes, node)
		}
	}

	// Sort nodes by distance to target
	sort.Slice(nodes, func(i, j int) bool {
		distI := distances[string(nodes[i].ID)]
		distJ := distances[string(nodes[j].ID)]
		return bytes.Compare(distI, distJ) < 0
	})

	// Return k closest nodes
	if len(nodes) > k {
		return nodes[:k]
	}
	return nodes
}

// ping checks if a node is still alive
func (dht *DHTNode) ping(node Node) bool {
	conn, err := net.DialTimeout("tcp", node.Address, time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Send ping request
	request := DHTRequest{
		Type: REQUEST_PING,
	}

	err = json.NewEncoder(conn).Encode(request)
	if err != nil {
		return false
	}

	// Wait for pong response
	var response DHTRequest
	err = json.NewDecoder(conn).Decode(&response)
	return err == nil && response.Type == REQUEST_PONG
}

// Bootstrap performs the Kademlia bootstrap process
func (dht *DHTNode) Bootstrap() error {
	log.Printf("[%s] Starting Kademlia bootstrap", dht.node.Address)

	// Load initial peers
	dht.mutex.RLock()
	var bootstrapNodes []Node
	for _, bucket := range dht.kBuckets {
		if bucket != nil {
			bootstrapNodes = append(bootstrapNodes, bucket.Nodes...)
		}
	}
	dht.mutex.RUnlock()

	// If no bootstrap nodes, return early
	if len(bootstrapNodes) == 0 {
		log.Printf("[%s] No bootstrap nodes available", dht.node.Address)
		return nil
	}

	// Keep track of queried nodes to avoid duplicates
	queriedNodes := make(map[string]bool)

	// Perform iterative node lookups
	for _, node := range bootstrapNodes {
		if node.Address == dht.node.Address {
			continue
		}

		if queriedNodes[node.Address] {
			continue
		}
		queriedNodes[node.Address] = true // Mark as queried

		log.Printf("[%s] Finding closest nodes to %x through %s",
			dht.node.Address, dht.node.ID, node.Address)

		// Connect to bootstrap node
		if err := dht.connectToPeer(node); err != nil {
			log.Printf("[%s] Failed to connect to bootstrap node %s: %v",
				dht.node.Address, node.Address, err)
			continue
		}

		// Add the bootstrap node to our k-buckets
		dht.addToKBucket(node)

		// Find nodes closest to our ID
		closestNodes := dht.findClosestNodes(dht.node.ID, NUMBER_OF_CLOSEST_PEERS)

		// Query each of the closest nodes
		for _, closeNode := range closestNodes {
			if queriedNodes[closeNode.Address] {
				continue
			}
			queriedNodes[closeNode.Address] = true // Mark as queried

			log.Printf("[%s] Querying close node %s", dht.node.Address, closeNode.Address)

			if err := dht.connectToPeer(closeNode); err != nil {
				log.Printf("[%s] Failed to connect to close node %s: %v",
					dht.node.Address, closeNode.Address, err)
				continue
			}

			// Add successful connections to k-buckets
			dht.addToKBucket(closeNode)
		}
	}

	log.Printf("[%s] Bootstrap complete. Total nodes queried: %d",
		dht.node.Address, len(queriedNodes))

	return nil
}

// connectToPeer connects to a peer and sends a DHT request to get their k-buckets
// Used for bootstrapping and node lookups
func (dht *DHTNode) connectToPeer(node Node) error {
	conn, err := net.DialTimeout("tcp", node.Address, time.Second*3)
	if err != nil {
		return fmt.Errorf("dial failed: %v", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(time.Second * 5))

	request := DHTRequest{
		Type: REQUEST_GET_DHT,
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

	for _, node := range response.Nodes {
		dht.addToKBucket(node)
	}

	// Update local DHT with unique entries
	dht.mutex.Lock()
	for k, v := range response.DataStore {
		dht.dataStore[k] = v
	}
	dht.mutex.Unlock()

	return nil
}

// addNode adds a new node to the k-buckets
func (dht *DHTNode) addNode(node Node) {
	dht.addToKBucket(node)
}

// hasNode checks if a node is already in the k-buckets
func (dht *DHTNode) hasNode(address string) bool {
	dht.mutex.RLock()
	defer dht.mutex.RUnlock()

	// Check all k-buckets for the node
	for _, bucket := range dht.kBuckets {
		for _, node := range bucket.Nodes {
			if node.Address == address {
				return true
			}
		}
	}
	return false
}

// getNode returns a node by its address if it exists in any k-bucket
func (dht *DHTNode) getNode(address string) (Node, bool) {
	dht.mutex.RLock()
	defer dht.mutex.RUnlock()

	for _, bucket := range dht.kBuckets {
		for _, node := range bucket.Nodes {
			if node.Address == address {
				return node, true
			}
		}
	}
	return Node{}, false
}

// removeNode removes a node from its k-bucket if it exists
func (dht *DHTNode) removeNode(address string) {
	dht.mutex.Lock()
	defer dht.mutex.Unlock()

	for i, bucket := range dht.kBuckets {
		for j, node := range bucket.Nodes {
			if node.Address == address {
				// Remove the node from the bucket
				dht.kBuckets[i].Nodes = append(bucket.Nodes[:j], bucket.Nodes[j+1:]...)
				return
			}
		}
	}
}

// handleGetDHT sends the k-buckets and data store to the requesting node
func (dht *DHTNode) handleGetDHT(conn net.Conn) error {
	var nodes []Node
	dht.mutex.RLock()

	// Add all nodes from k-buckets
	for _, bucket := range dht.kBuckets {
		nodes = append(nodes, bucket.Nodes...)
	}

	response := DHTResponse{
		Nodes:     nodes,
		DataStore: dht.dataStore,
	}
	dht.mutex.RUnlock()

	if err := json.NewEncoder(conn).Encode(response); err != nil {
		log.Printf("[%s] Error sending DHT response: %v", dht.node.Address, err)
		return err
	}

	log.Printf("[%s] DHT response sent", dht.node.Address)
	return nil
}

// handleStore stores a key-value pair in the local data store
func (dht *DHTNode) handleStore(request DHTRequest) error {
	if request.Payload == nil {
		return fmt.Errorf("empty payload")
	}

	// Assert the payload to map[string]string
	payload, ok := request.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload type: expected map[string]string, got %T", request.Payload)
	}

	// Extract key and value, converting to strings
	key, ok := payload["key"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid key in payload")
	}

	value, ok := payload["value"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid value in payload")
	}

	dht.mutex.Lock()
	dht.dataStore[key] = value
	dht.mutex.Unlock()

	log.Printf("[%s] Stored key-value pair: %s -> %s",
		dht.node.Address, key, value)
	return nil
}

// handleGet retrieves a value from the local data store
func (dht *DHTNode) handleGet(conn net.Conn, request DHTRequest) error {
	if payload, ok := request.Payload.(map[string]string); ok {
		key := payload["key"]

		// First check local storage
		dht.mutex.RLock()
		value, exists := dht.dataStore[key]
		dht.mutex.RUnlock()

		var response DHTResponse
		if !exists {
			// Key not found locally, find closest nodes
			log.Printf("[%s] Key '%s' not found locally, returning closest nodes",
				dht.node.Address, key)

			// Get hash of the key for finding closest nodes
			keyHash := sha3.Sum256([]byte(key))
			closestNodes := dht.findClosestNodes(keyHash[:], NUMBER_OF_CLOSEST_PEERS)

			// Filter out requesting node if it's in the closest nodes
			filteredNodes := make([]Node, 0)
			requestAddr := conn.RemoteAddr().String()
			for _, node := range closestNodes {
				if node.Address != requestAddr {
					filteredNodes = append(filteredNodes, node)
				}
			}

			response = DHTResponse{
				Nodes:     filteredNodes,
				DataStore: nil,
			}

			log.Printf("[%s] Returning %d closest nodes for key '%s'",
				dht.node.Address, len(filteredNodes), key)
		} else {
			// We found the value locally
			response = DHTResponse{
				Nodes: nil, // No need to include nodes when we have the value
				DataStore: map[string]string{
					key: value,
				},
			}
			log.Printf("[%s] Found value locally for key '%s'",
				dht.node.Address, key)
		}

		// Send the response
		if err := json.NewEncoder(conn).Encode(response); err != nil {
			log.Printf("[%s] Error sending GET response: %v",
				dht.node.Address, err)
			return err
		}

		log.Printf("[%s] GET response sent successfully", dht.node.Address)
		return nil
	}
	return fmt.Errorf("invalid payload format")
}

// handlePing responds to a PING request with a PONG
func (dht *DHTNode) handlePing(conn net.Conn) error {
	request := DHTRequest{
		Type: REQUEST_PONG,
	}

	if err := json.NewEncoder(conn).Encode(request); err != nil {
		log.Printf("[%s] Error sending PONG response: %v", dht.node.Address, err)
		return err
	}
	return nil
}

// loadPeersFromConfig loads a list of peers from a configuration file
func (dht *DHTNode) loadPeersFromConfig(peers map[string]Node) error {

	// Now add unique peers to routing table
	for _, node := range peers {
		log.Printf("[%s] Adding peer %s to routing table", dht.node.Address, node.Address)
		dht.addToKBucket(node)
	}

	return nil
}

// savePeersToConfig saves a list of peers to a configuration file
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

	for _, bucket := range dht.kBuckets {
		for _, node := range bucket.Nodes {
			_, err := file.WriteString(node.Address + "\n")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Store stores a key-value pair in the whole DHT
func (dht *DHTNode) Store(key, value string) error {
	// Find nodes closest to the key
	keyHash := sha3.Sum256([]byte(key))
	nodes := dht.findClosestNodes(keyHash[:], NUMBER_OF_CLOSEST_PEERS)

	// Store locally
	dht.mutex.Lock()
	dht.dataStore[key] = value
	dht.mutex.Unlock()

	// Store on closest nodes
	for _, node := range nodes {
		if node.Address == dht.node.Address {
			continue
		}

		// Connect to the node
		conn, err := net.DialTimeout("tcp", node.Address, time.Second*3)
		if err != nil {
			log.Printf("[%s] Failed to connect to node %s for storing: %v",
				dht.node.Address, node.Address, err)
			continue
		}
		defer conn.Close()

		// Create store request with properly typed payload
		request := DHTRequest{
			Type: REQUEST_STORE,
			Payload: map[string]string{
				"key":   key,
				"value": value,
			},
		}

		// Send the store request
		if err := json.NewEncoder(conn).Encode(request); err != nil {
			log.Printf("[%s] Failed to send store request to %s: %v",
				dht.node.Address, node.Address, err)
			continue
		}
	}

	return nil
}

// Get retrieves a value from the DHT
// Returns the value and a boolean indicating if the value was found
func (dht *DHTNode) Get(key string) (string, bool) {
	// Check local storage first
	dht.mutex.RLock()
	value, ok := dht.dataStore[key]
	dht.mutex.RUnlock()
	if ok {
		return value, true
	}

	// Keep track of queried nodes to avoid loops
	queriedNodes := make(map[string]bool)
	queriedNodes[dht.node.Address] = true

	// Find nodes closest to key
	hash := sha3.Sum256([]byte(key))
	nodesToQuery := dht.findClosestNodes(hash[:], NUMBER_OF_CLOSEST_PEERS)

	// Keep querying until we find the value or run out of nodes
	for len(nodesToQuery) > 0 {
		node := nodesToQuery[0]
		nodesToQuery = nodesToQuery[1:]

		// Skip if we've already queried this node
		if queriedNodes[node.Address] {
			continue
		}
		queriedNodes[node.Address] = true

		// Query the node
		conn, err := net.DialTimeout("tcp", node.Address, 3*time.Second)
		if err != nil {
			log.Printf("[%s] Failed to connect to node %s for getting: %v",
				dht.node.Address, node.Address, err)
			continue
		}

		request := DHTRequest{
			Type: REQUEST_GET,
			Payload: map[string]string{
				"key": key,
			},
		}

		if err := json.NewEncoder(conn).Encode(request); err != nil {
			conn.Close()
			continue
		}

		var response DHTResponse
		if err := json.NewDecoder(conn).Decode(&response); err != nil {
			conn.Close()
			continue
		}
		conn.Close()

		// Check if we got the value
		if response.DataStore != nil {
			if value, exists := response.DataStore[key]; exists {
				// Store it locally for future use
				dht.mutex.Lock()
				dht.dataStore[key] = value
				dht.mutex.Unlock()
				return value, true
			}
		}

		// If we got more nodes to query, add them to our list
		if response.Nodes != nil {
			for _, newNode := range response.Nodes {
				if !queriedNodes[newNode.Address] {
					nodesToQuery = append(nodesToQuery, newNode)
				}
			}
		}
	}

	return "", false
}

// RegisterUser registers a user in the DHT
func (dht *DHTNode) RegisterUser(userID string, address string, publicKey []byte) error {
	// Create user info
	userInfo := map[string]string{
		"userID":    userID,
		"address":   address,
		"status":    "online",
		"lastSeen":  time.Now().String(),
		"publicKey": string(publicKey),
	}

	// Convert to JSON
	infoBytes, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}

	// Store locally
	dht.mutex.Lock()
	dht.dataStore[userID] = string(infoBytes)
	dht.mutex.Unlock()

	// Store in DHT
	dht.Store(userID, string(infoBytes))

	return nil
}

// LookupUser looks up a user in the DHT
// Returns the user info as a map
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

// Print prints the DHT node's state
func (dht *DHTNode) Print() {
	fmt.Println("\n=== K-Buckets ===")
	for i, bucket := range dht.kBuckets {
		if len(bucket.Nodes) > 0 {
			fmt.Printf("Bucket %d:\n", i)
			for _, node := range bucket.Nodes {
				fmt.Printf("  %s -> %x\n", node.Address, node.ID)
			}
		}
	}

	fmt.Println("\n=== Data Store ===")
	fmt.Println("UserID -> UserInfo")
	for userID, data := range dht.dataStore {
		var userInfo map[string]string
		json.Unmarshal([]byte(data), &userInfo)
		fmt.Printf("%s -> %v\n", userID, userInfo)
	}
}
