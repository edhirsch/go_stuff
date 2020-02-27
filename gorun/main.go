package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
	"unicode/utf8"

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

// Command pre-defined struct
// ------------------------------------
type Command struct {
	Name        string `yaml:"name"`
	Command     string `yaml:"command"`
	Description string `yaml:"description"`
	Header      string `yaml:"header"`
	Output      string
	ReturnCode  int
}

// Node pre-defined struct
type Node struct {
	Client SSH
	Output string
}

// Nodes pre-defined struct
type Nodes []Node

// Filter yaml struct
type Filter struct {
	RegEx string
	Save  string
}

func printTabbedTable(lines []string) {
	writer := tabwriter.NewWriter(os.Stdout, 20, 8, 1, '\t', tabwriter.AlignRight)
	for i := 0; i < len(lines); i++ {
		fmt.Fprintln(writer, lines[i])
	}
	writer.Flush()
}

func getArgs() (string, string, string, error) {
	args := os.Args[1:]
	if len(args) < 2 {
		return "", "", "", errors.New("error: insufficient arguments")
	}
	return args[0], args[1], strings.Join(args[2:], " "), nil
}

func addDefaultBanner(command string, duration string, sshClient SSH) string {
	var banner string
	x := strings.Repeat("-", utf8.RuneCountInString(sshClient.Server)+utf8.RuneCountInString(sshClient.Port)+
		utf8.RuneCountInString(command)+utf8.RuneCountInString(duration)+9)
	banner = banner + fmt.Sprintf("%v\n", x)
	banner = banner + fmt.Sprintf("| %v:%v; %v; %v |\n", sshClient.Server, sshClient.Port, command, duration)
	banner = banner + fmt.Sprintf("%v\n", x)

	return banner
}

func runCommandOnHosts(command Command, sshClients Nodes) {
	var wg sync.WaitGroup

	for i := 0; i < len(sshClients); i++ {
		c := make(chan string)
		var output string
		t1 := time.Now()
		wg.Add(1)
		go runCommandParallel(command.Command, sshClients[i].Client, &wg, c)
		go func(sshClient *Node) {
			output = <-c
			t2 := time.Now()
			tdiff := t2.Sub(t1)
			duration := fmt.Sprintf("%0.2vs", tdiff.Seconds())
			if command.Header == "" {
				banner := addDefaultBanner(command.Command, duration, sshClient.Client)
				sshClient.Output = banner + output
				fmt.Printf("%v\n\n", sshClient.Output)
			} else {
				sshClient.Output = output
			}

		}(&sshClients[i])
	}
	wg.Wait()
	if command.Header != "" {
		outputs := getAllOutputs(sshClients)
		printOutputWithCustomBanner(command.Header, outputs)
	}
}

func getAllOutputs(sshClients Nodes) []string {
	var outputs []string
	for i := 0; i < len(sshClients); i++ {
		outputs = append(outputs, sshClients[i].Output)
	}
	return outputs
}

func runCommandParallel(command string, sshClient SSH, wg *sync.WaitGroup, c chan string) {
	defer wg.Done()
	err := sshClient.Connect(CertPassword)
	if err != nil {
		c <- fmt.Sprintln(err)
		return
	}
	sshClient.RefreshSession()
	commandOutput, err := sshClient.RunCommand(command)

	c <- commandOutput
	sshClient.Close()
}

func matchHost(hostString string, hostsList Nodes) (Nodes, error) {
	var foundHosts Nodes
	for i := 0; i < len(hostsList); i++ {
		matched, err := regexp.MatchString(hostString, hostsList[i].Client.Server)
		if err == nil {
			if matched {
				foundHosts = append(foundHosts, hostsList[i])
			}
		}
	}
	if len(foundHosts) == 0 {
		return foundHosts, fmt.Errorf("error: couldn't match any hosts using the provided pattern '%v'", hostString)
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
	return foundCommand, matchedPartial, fmt.Errorf("error: couldn't match any commands using the provided labels '%v'", commandString)
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

func showCommands(commands []Command) {
	var lines []string
	lines = append(lines, "LABELS\tCOMMAND\tDESCRIPTION")
	for i := 0; i < len(commands); i++ {
		line := fmt.Sprintf("%v\t%.35v\t%v", commands[i].Name, commands[i].Command, commands[i].Description)
		lines = append(lines, line)
	}
	printTabbedTable(lines)
}

func printOutputWithCustomBanner(banner string, output []string) {
	var lines []string
	banner = fmt.Sprintf(banner)
	lines = append(lines, banner)
	lines = append(lines, output...)
	printTabbedTable(lines)
}

func showHelp() {
	help := `Usage :
	ybssh <hosts> <command labels>
	ybssh <hosts> commands
	ybssh <hosts> nodes
	ybssh <hosts> exec <command>
	`
	fmt.Println(help)
}

func main() {

	var yamlHostsFile = "hosts_local.yaml"
	var yamlCommandsFile = "commands.yaml"
	var fullCommand string
	AuthType = CertPassword

	hosts, err := ReadHostsYamlFile(yamlHostsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	for i := 0; i < len(hosts); i++ {
		hosts[i].Client.init()
	}

	commands, err := ReadCommandsYamlFile(yamlCommandsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	firstArg, secondArg, otherArg, err := getArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		showHelp()
		return
	}
	fullCommand = secondArg + " " + otherArg

	matchedHosts, err := matchHost(firstArg, hosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	switch secondArg {
	case "exec":
		var execCommand Command
		execCommand.Command = otherArg
		runCommandOnHosts(execCommand, matchedHosts)

	case "commands":
		showCommands(commands)
		fmt.Println()
		break

	default:
		matchedCommand, partialCommands, err := matchCommand(fullCommand, commands)
		if err != nil {
			fmt.Printf("\nCouldn't match any command using labels '%v'. \n", fullCommand)
			fmt.Printf("Please check the '%v' file for the list of available commands. \n\n", yamlCommandsFile)
			fmt.Printf("For running one time commands, you can use :\n")
			fmt.Printf("ybssh --exec '%v'\n\n", fullCommand)
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}
		if len(partialCommands) > 0 {
			fmt.Printf("Matched the following command labels :\n\n")
			showCommands(partialCommands)
			fmt.Println()
			return
		}
		runCommandOnHosts(matchedCommand, matchedHosts)
	}
}
