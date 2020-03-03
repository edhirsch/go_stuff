package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// DefaultConfig global variables
var DefaultConfig SSHDefaults

// SSH yaml pre-defined structures
// ------------------------------------
type SSH struct {
	Server   string `yaml:"server"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	session  *ssh.Session
	client   *ssh.Client
}

// SSHDefaults pre-defined struct
// ------------------------------------
type SSHDefaults struct {
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func (sshClient *SSH) init() {
	fmt.Println(sshClient.Password)
	if sshClient.User == "" {
		sshClient.User = DefaultConfig.User
	}
	if sshClient.Port == "" {
		sshClient.Port = DefaultConfig.Port
	}
	if sshClient.Password == "" {
		sshClient.Password = DefaultConfig.Password
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
	if mode == CertPassword {
		auth = []ssh.AuthMethod{ssh.Password(sshClient.Password)}
	} else if mode == CertPublicKeyFile {
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
		Timeout: time.Second * DefaultTimeout,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", sshClient.Server, sshClient.Port), sshConfig)
	if err != nil {
		return err
	}
	sshClient.client = client
	return nil
}

// RunCommand function
func (sshClient *SSH) RunCommand(command string) (string, error) {
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
