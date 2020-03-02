package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
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
func (sshClient *SSH) RunCmd(cmd string) {
	out, err := sshClient.session.CombinedOutput(cmd)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(out))
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

// // Shell function
// func (sshClient *SSH) Shell() {
// 	modes := ssh.TerminalModes{
// 		ssh.ECHO:          0,     // disable echoing
// 		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
// 		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
// 	}
// 	// Request pseudo terminal
// 	if err := sshClient.session.RequestPty("xterm", 40, 80, modes); err != nil {
// 		log.Fatal("request for pseudo terminal failed: ", err)
// 	}
// 	err := sshClient.session.Shell()
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// }

// Close function
func (sshClient *SSH) Close() {
	sshClient.session.Close()
	sshClient.client.Close()
}

// InteractiveShell function
func (sshClient *SSH) InteractiveShell() {
	cmd := ""
	for cmd != "exit\n" {
		fmt.Printf("[ %v@%v ]# ", sshClient.User, sshClient.IP)
		reader := bufio.NewReader(os.Stdin)
		cmd, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		sshClient.RunCmd(cmd)
		sshClient.RefreshSession()
	}
}

// main function
func main() {
	client := &SSH{
		IP:   "csaf-ast-ui.cisco.com",
		User: "root",
		Port: 22,
		Cert: "cisco123",
	}
	client.Connect(CertPassword)
	client.RefreshSession()
	client.InteractiveShell()
	//client.RunCmd("ls /home")
	client.Close()
}
