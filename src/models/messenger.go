package models

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
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
	listener net.Listener
	Ready    chan bool
	KeyPair  *KeyPair
}

func NewP2PMessenger(userID string, port string) *P2PMessenger {
	listener, err := net.Listen("tcp", "localhost:"+port)
	if err != nil {
		panic(err)
	}
	keyPair, err := GenerateKeyPair(2048)
	if err != nil {
		panic(err)
	}
	return &P2PMessenger{
		dht:      NewDHTNode("localhost:" + port),
		userID:   userID,
		listener: listener,
		Ready:    make(chan bool),
		KeyPair:  keyPair,
	}
}

func (p *P2PMessenger) Start() error {
	if p.listener == nil {
		return fmt.Errorf("listener not initialized")
	}

	publicKey, err := p.KeyPair.ExportPublicKey()
	if err != nil {
		return fmt.Errorf("failed to export public key: %v", err)
	}

	err = p.dht.RegisterUser(p.userID, p.listener.Addr().String(), publicKey)
	if err != nil {
		return fmt.Errorf("failed to register user: %v", err)
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

	log.Printf("[%s] Ready to accept connections", p.userID)
	p.Ready <- true
	time.Sleep(1 * time.Second) // Ready to be sure

	log.Printf("[%s] Starting bootstrap process", p.userID)
	err = p.dht.Bootstrap()
	if err != nil {
		return fmt.Errorf("Failed to bootstrap DHT: %v", err)
	}

	return nil
}

func (p *P2PMessenger) SendMessage(to, content string) error {
	userInfo, err := p.dht.LookupUser(to)
	if err != nil {
		return fmt.Errorf("user lookup failed: %v", err)
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

	conn, err := net.Dial("tcp", userInfo["address"])
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
