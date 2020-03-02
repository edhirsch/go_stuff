package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// Constants
const (
	CertPassword      = 1
	CertPublicKeyFile = 2
	DefaultTimeout    = 3 // second
)

// SSH function
type SSH struct {
	IP      string
	User    string
	Cert    string //password or key file path
	Port    int
	session *ssh.Session
	client  *ssh.Client
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
		auth = []ssh.AuthMethod{ssh.Password(sshClient.Cert)}
	} else if mode == CertPublicKeyFile {
		auth = []ssh.AuthMethod{sshClient.readPublicKeyFile(sshClient.Cert)}
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

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", sshClient.IP, sshClient.Port), sshConfig)
	if err != nil {
		fmt.Println(err)
		return
	}

	// session, err := client.NewSession()
	// if err != nil {
	// 	fmt.Println(err)
	// 	client.Close()
	// 	return
	// }

	// sshClient.session = session
	sshClient.client = client
}

// RunCmd function
func (sshClient *SSH) RunCmd(cmd string) string {
	out, err := sshClient.session.CombinedOutput(cmd)
	if err != nil {
		fmt.Println(err)
	}
	return string(out)
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

// InputCommand function
func InputCommand() string {
	cmd := ""
	fmt.Printf("[ user@hostname ]# ")
	reader := bufio.NewReader(os.Stdin)
	cmd, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return cmd
}

// ConnectAndRunCommandParallel function
func (sshClient *SSH) ConnectAndRunCommandParallel(cmd string, wg *sync.WaitGroup) {
	defer wg.Done()

	sshClient.Connect(CertPassword)
	sshClient.RefreshSession()
	output := sshClient.RunCmd(cmd)
	sshClient.Close()

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fmt.Printf("%v: %v\n", sshClient.IP, line)
	}
	fmt.Println()
}

// main function
func main() {
	clients := []SSH{
		{IP: "sprint-rtp-cent.cisco.com", User: "root", Port: 22, Cert: "Cisco1@#"},
		{IP: "sprint-rtp-cent2.cisco.com", User: "root", Port: 22, Cert: "Cisco1@#"},
		{IP: "sprint-rtp-cent3.cisco.com", User: "root", Port: 22, Cert: "Cisco1@#"},
	}
	command := ""
	var wg sync.WaitGroup

	for true {
		command = InputCommand()
		if command == "exit" {
			break
		}
		for i := 0; i < len(clients); i++ {
			wg.Add(1)
			go clients[i].ConnectAndRunCommandParallel(command, &wg)
		}
		wg.Wait()
	}
}
