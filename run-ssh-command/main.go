package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

//import "golang.org/x/crypto/ssh"

var server string
var user string
var password []byte

func main() {
	inputConnectionDetails()
	fmt.Println()
	fmt.Printf("server:    %v\n", server)
	fmt.Printf("user name: %v\n", user)
	fmt.Printf("password:  %v\n", string(password))

}

func inputConnectionDetails() {
	fmt.Printf("Server name: ")
	_, err := fmt.Scanln(&server)
	if err != nil {
		os.Exit(1)
	}

	fmt.Printf("User name:   ")
	_, err = fmt.Scanln(&user)
	if err != nil {
		os.Exit(1)
	}

	fmt.Printf("Password:    ")
	password, err = terminal.ReadPassword(0)
	fmt.Println()
}
