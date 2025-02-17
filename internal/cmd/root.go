package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "p2pmessenger",
	Short: "A simple P2P messenger",
	Long: `A simple P2P messenger that uses a Kademlia DHT to store and retrieve messages.
	To start the messenger, run the following command:
	$ p2pmessenger`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
