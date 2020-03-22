package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"golang.org/x/crypto/ssh"
)

// KeyFile ; generate via gokey
var KeyFile string

// SSH yaml pre-defined structures
// ------------------------------------
type SSH struct {
	Server    string `yaml:"server"`
	Port      string `yaml:"port"`
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
	Defaults  SSHDefaults
	session   *ssh.Session
	client    *ssh.Client
	sshConfig *ssh.ClientConfig
}

// SSHDefaults pre-defined struct
// ------------------------------------
type SSHDefaults struct {
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// Node pre-defined struct
// -----------------------
type Node struct {
	Client     SSH
	Output     string
	ReturnCode int
}

// Nodes pre-defined struct
type Nodes []Node

func (sshClient *SSH) initHosts() {
	if sshClient.User == "" {
		sshClient.User = sshClient.Defaults.User
	}
	if sshClient.Port == "" {
		sshClient.Port = sshClient.Defaults.Port
	}
	if sshClient.Password == "" {
		sshClient.Password = sshClient.Defaults.Password
	} else {
		var err error
		sshClient.Password, err = decrypt(KeyFile, sshClient.Password)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func (sshClient *SSH) readPublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

// Connect function
func (sshClient *SSH) Connect(mode int) error {

	var sshConfig *ssh.ClientConfig
	var auth []ssh.AuthMethod
	if mode == 1 {
		auth = []ssh.AuthMethod{ssh.Password(sshClient.Password)}
	} else if mode == 2 {
		auth = []ssh.AuthMethod{sshClient.readPublicKeyFile(sshClient.Password)}
	} else {
		err := errors.New(fmt.Sprintln("error: does not support mode: ", mode))
		return err
	}

	sshConfig = &ssh.ClientConfig{
		User: sshClient.User,
		Auth: auth,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: time.Duration(Config.SSHDefaultTimeout) * time.Second,
	}
	sshClient.sshConfig = sshConfig

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", sshClient.Server, sshClient.Port), sshConfig)
	if err != nil {
		return err
	}
	sshClient.client = client
	return nil
}

// RunCommand function
func (sshClient *SSH) RunCommand(command string, pipe string) (string, error) {
	if pipe != "" {
		go func() {
			w, err := sshClient.session.StdinPipe()
			defer w.Close()
			if err != nil {
				fmt.Println(err)
			}
			fmt.Fprint(w, pipe)
		}()
	}
	output, err := sshClient.session.CombinedOutput(command)
	if len(output) > 0 {
		output = output[:len(output)-1]
	}
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}

// RefreshSession function
func (sshClient *SSH) RefreshSession() {
	session, err := sshClient.client.NewSession()
	if err != nil {
		fmt.Println(err)
		sshClient.Close()
		return
	}
	sshClient.session = session
}

// Close function
func (sshClient *SSH) Close() {
	sshClient.session.Close()
	sshClient.client.Close()
}

// CopyFileToRemote function
func (sshClient *SSH) CopyFileToRemote(file string, remotePath string, permission string) {

	// Create a new SCP client
	client := scp.NewClient(sshClient.Server, sshClient.sshConfig)

	// Connect to the remote server
	err := client.Connect()
	if err != nil {
		fmt.Println("Couldn't establish a connection to the remote server ", err)
		return
	}

	// Open a file
	f, _ := os.Open(file)

	// Close client connection after the file has been copied
	defer client.Close()

	// Close the file after it has been copied
	defer f.Close()

	// Copy the file over
	// Usage: CopyFile(fileReader, remotePath, permission)

	err = client.CopyFile(f, remotePath, permission)

	if err != nil {
		fmt.Println("Error while copying file ", err)
	}
	sshClient.session.Close()
	sshClient.client.Close()
}
