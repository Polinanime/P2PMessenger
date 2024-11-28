package utils

import "fmt"

func ReadUserCredentials() (string, string) {
	var port, username string
	fmt.Println("Enter your username: ")
	fmt.Scanln(&username)
	fmt.Println("Enter your port: ")
	fmt.Scanln(&port)

	return username, port
}
