package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

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
func (sshClient *SSH) Connect(mode int) {

	var sshConfig *ssh.ClientConfig
	var auth []ssh.AuthMethod
	if mode == CertPassword {
		auth = []ssh.AuthMethod{ssh.Password(sshClient.Password)}
	} else if mode == CertPublicKeyFile {
		auth = []ssh.AuthMethod{sshClient.readPublicKeyFile(sshClient.Password)}
	} else {
		log.Println("does not support mode: ", mode)
		return
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
		fmt.Println(err)
		return
	}
	sshClient.client = client
}

// RunCmd function
func (sshClient *SSH) RunCmd(command string) string {
	output, err := sshClient.session.CombinedOutput(command)
	if err != nil {
		fmt.Println(err)
	}
	if len(output) > 0 {
		output = output[:len(output)-1]
	}
	return string(output)
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
