package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

type server struct {
	hostname string
	username string
	password string
	port     string
}

var s server

func main() {

	s.port = "22"
	inputConnectionDetails()
	fmt.Println()
	fmt.Printf("server name: %v\n", s.hostname)
	fmt.Printf("user name:   %v\n", s.username)
	fmt.Printf("password:    %v\n", string(s.password))
	fmt.Println()
	fmt.Printf("Connecting to %v@%v:%v ..\n", s.username, s.hostname, s.port)

	conn, err := createSSHConnection(s.hostname, s.username, s.password, s.port)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(conn)
	fmt.Printf("%T\n", conn)

}

func inputConnectionDetails() {
	fmt.Printf("server name: ")
	_, err := fmt.Scanln(&s.hostname)
	if err != nil {
		os.Exit(1)
	}

	fmt.Printf("user name:   ")
	_, err = fmt.Scanln(&s.username)
	if err != nil {
		os.Exit(1)
	}

	fmt.Printf("password:    ")
	passwordB, err := terminal.ReadPassword(0)
	s.password = string(passwordB)
	fmt.Println()
}

func createSSHConnection(hostname string, username string, password string, port string) (interface{}, error) {
	// Configure ssh client
	sshConfig := &ssh.ClientConfig{
		User: s.username,
		Auth: []ssh.AuthMethod{ssh.Password(string(s.password))},
	}

	// Create ssh connection
	connection, err := ssh.Dial("tcp", s.hostname+":"+s.port, sshConfig)
	if err != nil {
		fmt.Println("Connection failed ..")
		return nil, fmt.Errorf("Failed to dial: %s", err)
	}
	fmt.Println("Connection succeeded !")

	return connection, nil
}
