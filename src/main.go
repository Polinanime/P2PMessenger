package main

import (
	"log"
	"os"

	"github.com/polinanime/p2pmessenger/cmd"
)

func main() {
	logFile, err := os.OpenFile("p2pmessenger.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Set log output to the file
	log.SetOutput(logFile)

	cmd.Execute()
}
