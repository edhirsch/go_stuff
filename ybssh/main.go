package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v2"

	"golang.org/x/crypto/ssh"
)

// Constants
const (
	CertPassword           = 1
	CertPublicKeyFile      = 2
	DefaultTimeout         = 3 // second
	Debug             bool = false
)

// AuthType ; CertPassword || CertPublicKeyFile
var AuthType int

// Command yaml pre-defined struct
// ------------------------------------
type Command struct {
	Name        string   `yaml:"name"`
	Command     string   `yaml:"command"`
	Description string   `yaml:"description"`
	Filters     []Filter `yaml:",flow"`
}

// Filter yaml struct
type Filter struct {
	RegEx string
	Save  string
}

// Script yaml pre-defined struct
// ------------------------------------
type Script struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
	Loop    Loop   `yaml:",flow"`
	Next    []Next `yaml:",flow"`
	Output  Output
}

// Next yaml pre-defined struct
type Next struct {
	Condition string
	Run       string
}

// Loop yaml struct
type Loop struct {
	Repeat int
	Break  string
	Next   string
}

// Output yaml struct
type Output struct {
	Stdout   string
	Stderr   string
	Variable struct {
		Name  string
		Value string
	}
}

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

// MultiSSH pre-defined structure
type MultiSSH []SSH

// ReadScriptYamlFile function
func ReadScriptYamlFile(fileName string) ([]Script, error) {

	var yamlConfig []Script
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

	if Debug {
		fmt.Println(yamlConfig)
	}

	return yamlConfig, nil
}

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

	if Debug {
		fmt.Println(yamlConfig)
	}

	return yamlConfig, nil
}

// ReadHostsYamlFile function
func ReadHostsYamlFile(fileName string) (MultiSSH, error) {

	var yamlConfig MultiSSH
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

	if Debug {
		fmt.Println(yamlConfig)
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

// ConnectAndRunCommandParallel function
func (sshClient *SSH) ConnectAndRunCommandParallel(cmd string, wg *sync.WaitGroup) {
	defer wg.Done()

	sshClient.Connect(AuthType)
	sshClient.RefreshSession()
	output := sshClient.RunCmd(cmd)
	sshClient.Close()

	lines := strings.Split(output, "\n")
	x := strings.Repeat("-", utf8.RuneCountInString(sshClient.Server)+utf8.RuneCountInString(sshClient.Port)+5)
	fmt.Printf("%v\n", x)
	fmt.Printf("| %v:%v |\n", sshClient.Server, sshClient.Port)
	fmt.Printf("%v\n", x)
	for _, line := range lines {
		fmt.Printf("%v\n", line)
	}
}

// RunCommandOnHosts function
func (sshClients MultiSSH) RunCommandOnHosts(command string) {
	var wg sync.WaitGroup
	for i := 0; i < len(sshClients); i++ {
		wg.Add(1)
		go sshClients[i].ConnectAndRunCommandParallel(command, &wg)
	}
	wg.Wait()
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

func getArgs() (string, string, string, error) {
	args := os.Args[1:]
	if len(args) < 2 {
		return "", "", "", errors.New("error: insufficient arguments")
	}
	return args[0], args[1], strings.Join(args[2:], " "), nil
}

func matchHost(hostString string, hostsList MultiSSH) (MultiSSH, error) {
	var foundHosts MultiSSH
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

func matchCommand(commandString string, commandList []Command) (Command, []Command, error) {
	var foundCommand Command
	var matchedPartial []Command
	commandLabels := strings.Fields(commandString)
	sort.Strings(commandLabels)
	for i := 0; i < len(commandList); i++ {
		commandListLabels := strings.Fields(commandList[i].Name)
		sort.Strings(commandListLabels)
		if reflect.DeepEqual(commandListLabels, commandLabels) == true {
			foundCommand = commandList[i]
		} else {
			if matchArrayInArray(commandLabels, commandListLabels) == true {
				matchedPartial = append(matchedPartial, commandList[i])
			}
		}
	}
	if foundCommand.Name != "" || len(matchedPartial) > 0 {
		return foundCommand, matchedPartial, nil
	}
	return foundCommand, matchedPartial, errors.New("error: match failed")
}

func matchArrayInArray(array1 []string, array2 []string) bool {
	matched := 0
	for i := 0; i < len(array1); i++ {
		for j := 0; j < len(array2); j++ {
			if array1[i] == array2[j] {
				matched++
				continue
			}
		}
	}
	if matched != len(array1) {
		return false
	}
	return true
}

func showHelp() {
	help := `Usage :
	ybssh <hosts> <command labels>
	ybssh <hosts> --exec <command>
	ybssh <hosts> --script <script.yaml>
	`
	fmt.Println(help)
}

func main() {

	var yamlHosts = "hosts.yaml"
	var yamlCommands = "commands.yaml"
	var fullCommand string
	var execCommand string
	AuthType = CertPassword

	hosts, err := ReadHostsYamlFile(yamlHosts)
	if err != nil {
		os.Exit(1)
	}

	commands, err := ReadCommandsYamlFile(yamlCommands)
	if err != nil {
		os.Exit(2)
	}

	hostArg, commandArg, argsArg, err := getArgs()
	if err != nil {
		showHelp()
		fmt.Println(err)
		os.Exit(3)
	}
	fullCommand = commandArg + " " + argsArg

	matchedHosts, err := matchHost(hostArg, hosts)
	if err != nil {
		os.Exit(4)
	}

	switch commandArg {
	case "--exec":
		execCommand = argsArg
		matchedHosts.RunCommandOnHosts(execCommand)
	case "--apply":
		yamlScript := argsArg
		script, err := ReadScriptYamlFile(yamlScript)
		if err != nil {
			os.Exit(5)
		}
		fmt.Println(script)
	case "--list":
		fmt.Println("Placeholder")
	default:
		matchedCommand, partialCommands, err := matchCommand(fullCommand, commands)
		if err != nil {
			fmt.Printf("\nCouldn't match any command using labels '%v'. \n", fullCommand)
			fmt.Printf("Please check the '%v' file for the list of available commands. \n\n", yamlCommands)
			fmt.Printf("For running one time commands, you can use :\n")
			fmt.Printf("ybssh --exec '%v'\n\n", fullCommand)
			break
		}
		if len(partialCommands) > 0 {
			fmt.Printf("Did you mean ..\n")
			for i := 0; i < len(partialCommands); i++ {
				fmt.Printf("\t%v\t%v\t%v\n", partialCommands[i].Name, partialCommands[i].Command, partialCommands[i].Description)
			}
			fmt.Println()
			break
		}
		execCommand = matchedCommand.Command
		matchedHosts.RunCommandOnHosts(execCommand)
	}
}
