package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/polinanime/p2pmessenger/internal/models"
	"github.com/polinanime/p2pmessenger/internal/utils"
	"github.com/spf13/cobra"
)

var (
	userID string
	port   string
)

var startCmd = &cobra.Command{
	Use:   "start [userID] [port]",
	Short: "Start the P2P messenger",
	Args:  cobra.ExactArgs(2),
	Run:   startMessenger,
}

func init() {
	rootCmd.AddCommand(startCmd)
	// startCmd.Flags().StringVarP(&userID, "user", "u", "", "User ID for the messenger")
	// startCmd.Flags().StringVarP(&port, "port", "p", "", "Port to listen")
	// startCmd.MarkFlagRequired("user")
	// startCmd.MarkFlagRequired("port")
	// rootCmd.AddCommand(startCmd)
}

func startMessenger(cmd *cobra.Command, args []string) {
	userID := args[0]
	port := args[1]
	configPath := ".config/peers.txt"
	settings := utils.NewSettings(userID, port, configPath)

	messenger := models.NewP2PMessenger(*settings)
	defer messenger.Close()

	if err := messenger.Start(); err != nil {
		panic(err)
	}

	fmt.Printf("\n=== P2P Messenger ===\n")
	fmt.Printf("User: %s\n", userID)
	fmt.Printf("Port: %s\n", port)
	fmt.Printf("Type 'help' for available commands\n\n")

	// Simple CLI loop
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		// Trim spaces and newline
		input = strings.TrimSpace(input)

		// Split the input into command and arguments
		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}

		command := strings.ToLower(args[0])
		switch command {
		case "help":
			printHelp()
		case "send":
			if len(args) < 3 {
				fmt.Println("\nUsage: send <user> <message>")
				continue
			}
			recipient := args[1]
			message := strings.Join(args[2:], " ")
			if err := messenger.SendMessage(recipient, message); err != nil {
				fmt.Printf("\nError sending message: %v\n", err)
			} else {
				fmt.Printf("\nMessage sent to %s\n", recipient)
			}
		case "list":
			messenger.PrintDht()
		case "history":
			if len(args) > 1 {
				showUserHistory(messenger, args[1])
			} else {
				showAllHistory(messenger)
			}
		case "scan":
			fmt.Println("\nScanning for new messages...")
			messenger.ScanPeers()
		case "whoami":
			printWhoAmI(messenger)
		case "users":
			fmt.Println("\nRegistered Users:")
			messenger.PrintUsers()
		case "add":
			if len(args) < 2 {
				fmt.Println("\nUsage: add <address>")
				continue
			}
			address := args[1]
			if err := messenger.AddPeer(address); err != nil {
				fmt.Printf("\nError adding user: %v\n", err)
			} else {
				fmt.Printf("\nUser added: %s\n", address)
			}
		case "exit":
			fmt.Println("\nExiting...")
			return
		default:
			fmt.Printf("\nUnknown command '%s'. Type 'help' for available commands\n", command)
		}
	}
}

func showUserHistory(messenger *models.P2PMessenger, user string) {
	messages := messenger.GetHistory()
	fmt.Printf("\nChat history with %s:\n", user)
	for _, msg := range messages {
		if (msg.From == user && msg.To == messenger.GetUserID()) ||
			(msg.From == messenger.GetUserID() && msg.To == user) {
			fmt.Printf("[%s] %s -> %s: %s\n",
				msg.Timestamp.Format("15:04:05"),
				msg.From,
				msg.To,
				msg.Content)
		}
	}
	fmt.Println()
}

func showAllHistory(messenger *models.P2PMessenger) {
	messages := messenger.GetHistory()
	fmt.Println("\nAll message history:")
	for _, msg := range messages {
		fmt.Printf("[%s] %s -> %s: %s\n",
			msg.Timestamp.Format("15:04:05"),
			msg.From,
			msg.To,
			msg.Content)
	}
	fmt.Println()
}

func scanPeers(messenger *models.P2PMessenger) {
	err := messenger.ScanPeers()
	if err != nil {
		fmt.Printf("Error scanning peers: %v\n", err)
	}
}

func printWhoAmI(messenger *models.P2PMessenger) {
	userID := messenger.GetUserID()
	fmt.Printf("User ID: %s\n", userID)
	fmt.Printf("Address: %s\n", models.GetContainerIP())
}

func printHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  help                    Show this help message")
	fmt.Println("  send <user> <message>   Send a message to a user")
	fmt.Println("  list                    List known peers and DHT state")
	fmt.Println("  history                 Show all message history")
	fmt.Println("  history <user>          Show chat history with specific user")
	fmt.Println("  exit                    Exit the application")
	fmt.Println("  scan                    Scan for peers in the network")
	fmt.Println("  whoami                  Print the user data")
	fmt.Println("  users                   Print all registered users")
	fmt.Println("  add <address>           Add user manually to DHT")
	fmt.Println("\nExample:")
	fmt.Println("  > send Eve Hello, how are you?")
	fmt.Println("  > history Eve")
	fmt.Println("")
}
