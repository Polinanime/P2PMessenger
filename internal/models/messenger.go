package models

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/polinanime/p2pmessenger/internal/mocks"
	"github.com/polinanime/p2pmessenger/internal/utils"
)

type Message struct {
	From             string
	To               string
	EncryptedContent []byte
	PublicKey        []byte
	FromAddress      string
}

type P2PMessenger struct {
	dht      *DHTNode
	userID   string
	port     string
	listener net.Listener
	Ready    chan bool
	KeyPair  *KeyPair
	history  *MessageHistory
	testConn net.Conn
	testMode bool
}

// NewP2PMessenger creates a new P2PMessenger instance.
func NewP2PMessenger(settings utils.Settings) *P2PMessenger {
	address := fmt.Sprintf("0.0.0.0:%s", settings.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	keyPair, err := GenerateKeyPair(2048)
	if err != nil {
		panic(err)
	}

	containerIP := GetContainerIP()
	dhtAddress := fmt.Sprintf("%s:%s", containerIP, settings.Port)
	settings.Address = dhtAddress // Update settings with DHT address

	return &P2PMessenger{
		dht:      NewDHTNode(settings),
		userID:   settings.Username,
		port:     settings.Port,
		listener: listener,
		Ready:    make(chan bool),
		KeyPair:  keyPair,
		history:  NewMessageHistory(),
	}
}

// NewMockMessenger creates a new P2PMessenger instance.
func NewMockMessenger(settings utils.Settings, conn net.Conn) *P2PMessenger {
	keyPair, err := GenerateKeyPair(2048)
	if err != nil {
		panic(err)
	}

	containerIP := GetContainerIP()
	dhtAddress := fmt.Sprintf("%s:%s", containerIP, settings.Port)
	settings.Address = dhtAddress

	listener := &mocks.MockListener{
		Conn: conn,
	}

	return &P2PMessenger{
		dht:      NewDHTNode(settings),
		userID:   settings.Username,
		port:     settings.Port,
		listener: listener,
		Ready:    make(chan bool),
		KeyPair:  keyPair,
		history:  NewMessageHistory(),
		testConn: conn,
		testMode: true,
	}
}

// Start starts the P2PMessenger instance.
func (p *P2PMessenger) Start() error {
	if p.listener == nil {
		return fmt.Errorf("listener not initialized")
	}

	// Export public key for registration
	publicKeyPEM, err := p.KeyPair.ExportPublicKey()
	if err != nil {
		return fmt.Errorf("failed to export public key: %v", err)
	}

	containerIP := GetContainerIP()
	address := fmt.Sprintf("%s:%s", containerIP, p.port)

	log.Printf("[%s] Registering with address: %s", p.userID, address)

	// Register user in DHT
	err = p.dht.RegisterUser(p.userID, address, publicKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to register user: %v", err)
	}

	// Bootstrap the DHT
	err = p.dht.Bootstrap()
	if err != nil {
		return fmt.Errorf("failed to bootstrap DHT: %v", err)
	}

	// Start listening for incoming messages
	go func() {
		log.Printf("Starting listener for %s on %s", p.userID, p.listener.Addr())
		for {
			conn, err := p.listener.Accept()
			if err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") {
					log.Printf("Error accepting connection: %v", err)
				}
				return
			}
			go p.handleIncomingConnection(conn)
		}
	}()

	return nil
}

// PrintUsers prints the list of registered users
func (p *P2PMessenger) PrintUsers() {
	p.dht.mutex.RLock()
	defer p.dht.mutex.RUnlock()

	fmt.Println("\n=== Registered Users ===")
	for _, value := range p.dht.dataStore {
		var userInfo map[string]string
		if err := json.Unmarshal([]byte(value), &userInfo); err == nil {
			if _, ok := userInfo["userID"]; ok {
				fmt.Printf("UserID: %s\n", userInfo["userID"])
				fmt.Printf("Address: %s\n", userInfo["address"])
				fmt.Printf("Status: %s\n", userInfo["status"])
				fmt.Printf("Last Seen: %s\n\n", userInfo["lastSeen"])
			}
		}
	}
}

// ScanPeers scans the network for peers
func (p *P2PMessenger) ScanPeers() error {
	return p.dht.Bootstrap()
}

// GetAddress returns the address of the P2PMessenger instance
func (p *P2PMessenger) GetAddress() string {
	return GetContainerIP() + ":" + p.port
}

// SendMessage sends encrypted message to a user
func (p *P2PMessenger) SendMessage(to, content string) error {
	if to == p.userID {
		return fmt.Errorf("cannot send message to myself")
	}

	userInfo, err := p.dht.LookupUser(to)
	if err != nil {
		return fmt.Errorf("user lookup failed: %v", err)
	}

	address := userInfo["address"]
	if address == "" {
		return fmt.Errorf("invalid address for user %s", to)
	}

	recipientPubKey, err := ImportPublicKey([]byte(userInfo["publicKey"]))
	if err != nil {
		return fmt.Errorf("failed to import recipient public key: %v", err)
	}

	// Encrypt the message content
	encryptedContent, err := Encrypt(recipientPubKey, []byte(content))
	if err != nil {
		return fmt.Errorf("encryption failed: %v", err)
	}

	// Get our public key
	publicKeyPEM, err := p.KeyPair.ExportPublicKey()
	if err != nil {
		return fmt.Errorf("failed to export public key: %v", err)
	}

	log.Printf("[%s] Sending message to %s at %s", p.userID, to, userInfo["address"])

	var conn net.Conn
	if p.testMode {
		conn = p.testConn
	} else {
		conn, err = net.Dial("tcp", address)
		if err != nil {
			return fmt.Errorf("connection failed: %v", err)
		}
		defer conn.Close()
	}
	message := Message{
		From:             p.userID,
		To:               to,
		EncryptedContent: encryptedContent,
		PublicKey:        publicKeyPEM,
		FromAddress:      GetContainerIP() + ":" + p.port,
	}

	// Debug logging
	messageBytes, _ := json.Marshal(message)
	log.Printf("[%s] DEBUG: Sending message bytes: %s", p.userID, string(messageBytes))

	if err := json.NewEncoder(conn).Encode(message); err != nil {
		return fmt.Errorf("message send failed: %v", err)
	}

	log.Printf("[%s] Message sent successfully to %s", p.userID, to)

	// Saving message to history (Only after successful send)
	go p.history.AddMessage(ChatMessage{
		To:        to,
		From:      p.userID,
		Content:   content,
		Timestamp: time.Now(),
	})

	return nil
}

func (p *P2PMessenger) handleIncomingConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Read until we get a complete JSON message
	data, err := reader.ReadBytes('\n')
	if err != nil {
		log.Printf("[%s] Error reading from connection: %v", p.userID, err)
		return
	}

	log.Printf("[%s] New connection from %s, received %d bytes",
		p.userID, conn.RemoteAddr(), len(data))

	// Try to decode as DHT request first
	var dhtRequest DHTRequest
	if err := json.Unmarshal(data, &dhtRequest); err == nil &&
		(dhtRequest.Type == REQUEST_GET ||
			dhtRequest.Type == REQUEST_STORE ||
			dhtRequest.Type == REQUEST_GET_DHT) {
		// Handle DHT request directly instead of passing the connection
		if err := p.handleDHTRequest(dhtRequest, conn); err != nil {
			log.Printf("[%s] Error handling DHT request: %v", p.userID, err)
		}
		return
	}

	// If not a DHT request, try to decode as message
	var message Message
	if err := json.Unmarshal(data, &message); err == nil {
		p.handleMessage(message)
		return
	}

	log.Printf("[%s] Could not decode incoming data as either DHT request or message",
		p.userID)
}

// HandleDHTRequest handles incoming DHT requests
// GET_DHT : Returns the k-buckets and data store
// STORE   : Stores a key-value pair locally
// GET	   : Retrieves a value from the local data store
// PING    : Responds with a PONG
func (p *P2PMessenger) handleDHTRequest(request DHTRequest, conn net.Conn) error {
	log.Printf("[%s] Handling DHT request type: %s", p.userID, request.Type)

	switch request.Type {
	case REQUEST_GET_DHT:
		return p.dht.handleGetDHT(conn)
	case REQUEST_STORE:
		return p.dht.handleStore(request)
	case REQUEST_GET:
		return p.dht.handleGet(conn, request)
	case REQUEST_PING:
		return p.dht.handlePing(conn)
	}

	return fmt.Errorf("unknown request type: %s", request.Type)
}

// AddPeer adds a new peer to the DHT
func (p *P2PMessenger) AddPeer(address string) error {
	return p.dht.AddPeer(address)
}

func (p *P2PMessenger) handleMessage(message Message) {

	// Check if we know the sender
	_, err := p.dht.LookupUser(message.From)
	if err != nil {
		// User not found in our DHT
		log.Printf("[%s] New user discovered: %s, adding to DHT", p.userID, message.From)

		// Get sender's address from the connection
		senderAddr := message.FromAddress

		// Register the new user in our DHT
		err = p.dht.RegisterUser(message.From, senderAddr, message.PublicKey)
		if err != nil {
			log.Printf("[%s] Failed to add new user to DHT: %v", p.userID, err)
		} else {
			log.Printf("[%s] Successfully added user %s to DHT", p.userID, message.From)
		}
	}

	decryptedContent, err := p.KeyPair.Decrypt(message.EncryptedContent)

	if err != nil {
		log.Printf("[%s] Error decrypting message: %v", p.userID, err)
		return
	}

	// Saving message to history (after successful decryption)
	go p.history.AddMessage(ChatMessage{
		From:      message.From,
		To:        message.To,
		Content:   string(decryptedContent), // Save decrypted content
		Timestamp: time.Now(),
	})

	// Debug logging
	log.Printf("[%s] Received message from %s to %s", p.userID, message.From, message.To)

	// Print message to console
	fmt.Printf("\n=== New Message ===\nFrom: %s\nTo: %s\nContent: %s\n\n",
		message.From, message.To, string(decryptedContent))
}

// Close closes the P2PMessenger instance.
func (p *P2PMessenger) Close() {
	p.listener.Close()
}

// PrintDht prints the DHTNode instance associated with the P2PMessenger.
func (p *P2PMessenger) PrintDht() {
	p.dht.Print()
}

// GetUserID returns the user ID of the P2PMessenger.
func (p *P2PMessenger) GetUserID() string {
	return p.userID
}

// GetHistory returns the chat history of the P2PMessenger.
func (p *P2PMessenger) GetHistory() []ChatMessage {
	p.history.mutex.RLock()
	defer p.history.mutex.RUnlock()
	return p.history.GetMessages()
}

// GetDHT returns the DHTNode instance associated with the P2PMessenger.
func (p *P2PMessenger) GetDHT() *DHTNode {
	return p.dht
}

// GetContainerIP returns the IP of the container
func GetContainerIP() string {
	// Default to Docker network IP if available
	ifaces, err := net.Interfaces()
	if err != nil {
		return "172.18.0.1" // fallback
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
					return ipnet.IP.String()
				}
			}
		}
	}

	return "172.18.0.1" // fallback
}
