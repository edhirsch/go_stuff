package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

//import "golang.org/x/crypto/ssh"
type Server struct {
	hostname string
	username string
	password []byte
}

var s Server

func main() {
	inputConnectionDetails()
	fmt.Println()
	fmt.Printf("server:    %v\n", s.hostname)
	fmt.Printf("user name: %v\n", s.username)
	fmt.Printf("password:  %v\n", string(s.password))

}

func inputConnectionDetails() {
	fmt.Printf("Server name: ")
	_, err := fmt.Scanln(&s.hostname)
	if err != nil {
		os.Exit(1)
	}

	fmt.Printf("User name:   ")
	_, err = fmt.Scanln(&s.username)
	if err != nil {
		os.Exit(1)
	}

	fmt.Printf("Password:    ")
	s.password, err = terminal.ReadPassword(0)
	fmt.Println()
}
