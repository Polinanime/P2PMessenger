package models

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/polinanime/p2pmessenger/utils"
)

type Message struct {
	From             string
	To               string
	EncryptedContent []byte
	PublicKey        []byte
}

type P2PMessenger struct {
	dht      *DHTNode
	userID   string
	port     string
	listener net.Listener
	Ready    chan bool
	KeyPair  *KeyPair
	history  *MessageHistory
}

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

	containerIP := getContainerIP()
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

func (p *P2PMessenger) Start() error {
	if p.listener == nil {
		return fmt.Errorf("listener not initialized")
	}

	// Export public key for registration
	publicKeyPEM, err := p.KeyPair.ExportPublicKey()
	if err != nil {
		return fmt.Errorf("failed to export public key: %v", err)
	}

	containerIP := getContainerIP()
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

func (p *P2PMessenger) ScanPeers() error {
	return p.dht.Bootstrap()
}

func (p *P2PMessenger) GetAddress() string {
	return p.listener.Addr().String()
}

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

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}
	defer conn.Close()

	message := Message{
		From:             p.userID,
		To:               to,
		EncryptedContent: encryptedContent,
		PublicKey:        publicKeyPEM,
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

	// Read all data from connection first
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Printf("[%s] Error reading from connection: %v", p.userID, err)
		return
	}
	data := buf[:n]

	log.Printf("[%s] New connection from %s, received %d bytes", p.userID, conn.RemoteAddr(), n)

	// Try to decode as DHT request first
	var dhtRequest DHTRequest
	if err := json.Unmarshal(data, &dhtRequest); err == nil && dhtRequest.Type == REQUEST_GET || dhtRequest.Type == REQUEST_STORE || dhtRequest.Type == REQUEST_GET_DHT {
		p.handleDHTRequest(conn)
		return
	}

	// If not a DHT request, try to decode as message
	var message Message
	if err := json.Unmarshal(data, &message); err == nil {
		p.handleMessage(message)
		return
	}

	log.Printf("[%s] Could not decode incoming data as either DHT request or message", p.userID)
}

func (p *P2PMessenger) handleDHTRequest(conn net.Conn) {
	log.Printf("[%s] Handling DHT request", p.userID)

	response := DHTResponse{
		DataStore: p.dht.dataStore,
	}

	if err := json.NewEncoder(conn).Encode(response); err != nil {
		log.Printf("[%s] Error sending DHT response: %v", p.userID, err)
		return
	}

	log.Printf("[%s] DHT response sent", p.userID)
}

func (p *P2PMessenger) handleMessage(message Message) {
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

	log.Printf("[%s] Received message from %s to %s", p.userID, message.From, message.To)
	fmt.Printf("\n=== New Message ===\nFrom: %s\nTo: %s\nContent: %s\n\n",
		message.From, message.To, string(decryptedContent))
}

func (p *P2PMessenger) Close() {
	p.listener.Close()
}

func (p *P2PMessenger) PrintDht() {
	p.dht.Print()
}

func (p *P2PMessenger) GetUserID() string {
	return p.userID
}

func (p *P2PMessenger) GetHistory() []ChatMessage {
	p.history.mutex.RLock()
	defer p.history.mutex.RUnlock()
	return p.history.GetMessages()
}

func getContainerIP() string {
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
