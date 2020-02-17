package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v2"

	"golang.org/x/crypto/ssh"
)

// Constants
const (
	CertPassword      = 1
	CertPublicKeyFile = 2
	DefaultTimeout    = 3 // second
)

// AuthType ; CertPassword || CertPublicKeyFile
var AuthType int

// Command yaml pre-defined struct
// ------------------------------------
type Command struct {
	Name       string   `yaml:"name"`
	Command    string   `yaml:"command"`
	Parameters []string `yaml:"parameters"`
	Depends    string   `yaml:"depends"`
	Output     Output   `yaml:",flow"`
	Flow       Flow     `yaml:",flow"`
}

// Flow yaml struct
type Flow struct {
	Sequential bool
	Parallel   bool
	Selection  string
	Iteration  string
}

// Output yaml struct
type Output struct {
	Raw      string
	Filter   string
	Filtered string
	Variable string
}

// ------------------------------------

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

// ------------------------------------

// ReadCommandsYamlFile function
func ReadCommandsYamlFile(fileName string) ([]Command, error) {

	var yamlConfig []Command
	yamlFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return yamlConfig, err
	}

	err = yaml.Unmarshal([]byte(yamlFile), &yamlConfig)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
		return yamlConfig, err
	}

	return yamlConfig, nil
}

// ReadHostsYamlFile function
func ReadHostsYamlFile(fileName string) ([]SSH, error) {

	var yamlConfig []SSH
	yamlFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return yamlConfig, err
	}

	err = yaml.Unmarshal([]byte(yamlFile), &yamlConfig)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
		return yamlConfig, err
	}

	return yamlConfig, nil
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

	sshClient.Connect(AuthType)
	sshClient.RefreshSession()
	output := sshClient.RunCmd(cmd)
	sshClient.Close()

	lines := strings.Split(output, "\n")
	x := strings.Repeat("-", utf8.RuneCountInString(sshClient.Server)+4)
	fmt.Printf("%v\n", x)
	fmt.Printf("| %v |\n", sshClient.Server)
	fmt.Printf("%v\n", x)
	for _, line := range lines {
		fmt.Printf("%v\n", line)
	}
}

func getArgs() (string, string, string, error) {
	args := os.Args[1:]
	if len(args) < 2 {
		return "", "", "", errors.New("error: insufficient arguments")
	}
	return args[0], args[1], strings.Join(args[2:], " "), nil
}

func matchHost(hostString string, hostsList []SSH) ([]SSH, error) {
	var foundHosts []SSH
	for i := 0; i < len(hostsList); i++ {
		matched, err := regexp.MatchString(hostString, hostsList[i].Server)
		if err == nil {
			if matched {
				foundHosts = append(foundHosts, hostsList[i])
			}
		}
	}
	if len(foundHosts) == 0 {
		return nil, errors.New("error: match failed")
	}
	return foundHosts, nil
}

func matchCommand(commandString string, commandList []Command) (Command, error) {
	var foundCommand Command
	for i := 0; i < len(commandList); i++ {
		if commandList[i].Name == commandString {
			foundCommand = commandList[i]
			return foundCommand, nil
		}
	}
	return foundCommand, errors.New("error: match failed")
}

func main() {

	var yamlHosts = "hosts.yaml"
	var yamlCommands = "commands.yaml"
	var wg sync.WaitGroup
	var execCommand string
	var matchedCommand Command
	AuthType = CertPassword

	hosts, err := ReadHostsYamlFile(yamlHosts)
	if err != nil {
		os.Exit(1)
	}
	fmt.Printf("%v\n", hosts)

	commands, err := ReadCommandsYamlFile(yamlCommands)
	if err != nil {
		os.Exit(2)
	}
	fmt.Printf("%v\n", commands)

	hostArg, commandArg, argsArg, err := getArgs()
	if err != nil {
		os.Exit(3)
	}

	matchedHosts, err := matchHost(hostArg, hosts)
	if err != nil {
		os.Exit(4)
	}

	if commandArg == "exec" {
		execCommand = argsArg
	} else {
		matchedCommand, err = matchCommand(commandArg, commands)
		if err != nil {
			os.Exit(5)
		}
	}

	for i := 0; i < len(matchedHosts); i++ {
		wg.Add(1)
		if execCommand != "" {
			go matchedHosts[i].ConnectAndRunCommandParallel(execCommand, &wg)
		} else {
			go matchedHosts[i].ConnectAndRunCommandParallel(matchedCommand.Command, &wg)
		}
	}
	wg.Wait()
}
