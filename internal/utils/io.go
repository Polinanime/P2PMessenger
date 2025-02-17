package utils

import "fmt"

func ReadUserCredentials() (string, string) {
	var port, username string
	fmt.Println("Enter your username: ")
	_, err := fmt.Scanln(&username)
	if err != nil {
		fmt.Println("Error reading username:", err)
		return "", ""
	}
	fmt.Println("Enter your port: ")
	_, err = fmt.Scanln(&port)
	if err != nil {
		fmt.Println("Error reading port:", err)
		return "", ""
	}
	if port == "" {
		port = "8080"
	}

	return username, port
}
