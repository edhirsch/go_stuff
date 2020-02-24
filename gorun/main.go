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
	"unicode/utf8"
)

// Constants
const (
	CertPassword           = 1
	CertPublicKeyFile      = 2
	DefaultTimeout         = 3 // second
	Debug             bool = false
)

// Banner ; true to show server name/command banner ; false to skip
var Banner = true

// AuthType ; CertPassword || CertPublicKeyFile
var AuthType int

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

// printTabbedTable function
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

// addDefaultBanner function
func (sshClient *SSH) addDefaultBanner(command string) string {
	var banner string
	x := strings.Repeat("-", utf8.RuneCountInString(sshClient.Server)+utf8.RuneCountInString(sshClient.Port)+
		utf8.RuneCountInString(command)+7)
	banner = banner + fmt.Sprintf("%v\n", x)
	banner = banner + fmt.Sprintf("| %v:%v; %v |\n", sshClient.Server, sshClient.Port, command)
	banner = banner + fmt.Sprintf("%v\n", x)

	return banner
}

// RunCommandOnHosts function
func (sshClients Nodes) RunCommandOnHosts(command string) {
	var wg sync.WaitGroup

	for i := 0; i < len(sshClients); i++ {
		ch := make(chan string)
		sshClients[i].Client.Connect(CertPassword)
		wg.Add(1)
		go sshClients[i].Client.RunCmdParallel(command, &wg, ch)
		output := <-ch
		if Banner == true {
			banner := sshClients[i].Client.addDefaultBanner(command)
			output = banner + output
		}
		sshClients[i].Output = output
	}
	wg.Wait()
}

// RunCmdParallel function
func (sshClient SSH) RunCmdParallel(command string, wg *sync.WaitGroup, c chan string) {
	defer wg.Done()
	sshClient.RefreshSession()
	commandOutput := sshClient.RunCmd(command)

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
		return foundHosts, errors.New("error: match failed")
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

func showCommands(commands []Command) {
	var lines []string
	lines = append(lines, "LABELS\tCOMMAND\tDESCRIPTION")
	for i := 0; i < len(commands); i++ {
		line := fmt.Sprintf("%v\t%v\t%v", commands[i].Name, commands[i].Command, commands[i].Description)
		lines = append(lines, line)
	}
	printTabbedTable(lines)
}

func printOutputWithCustomBanner(banner string, output []string) {
	Banner = false
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

	var yamlHosts = "hosts_local.yaml"
	var yamlCommands = "commands.yaml"
	var fullCommand string
	AuthType = CertPassword

	hosts, err := ReadHostsYamlFile(yamlHosts)
	if err != nil {
		os.Exit(1)
	}

	commands, err := ReadCommandsYamlFile(yamlCommands)
	if err != nil {
		os.Exit(2)
	}

	firstArg, secondArg, otherArg, err := getArgs()
	if err != nil {
		showHelp()
		fmt.Println(err)
		os.Exit(3)
	}
	fullCommand = secondArg + " " + otherArg

	matchedHosts, err := matchHost(firstArg, hosts)
	if err != nil {
		os.Exit(4)
	}

	switch secondArg {
	case "exec":
		matchedHosts.RunCommandOnHosts(otherArg)
		for i := 0; i < len(matchedHosts); i++ {
			fmt.Println(matchedHosts[i].Output)
		}

	case "commands":
		showCommands(commands)
		fmt.Println()
		break
	case "nodes":
		Banner = false
		var outputs []string

		matchedCommand, _, err := matchCommand("list nodes", commands)
		if err != nil {
			os.Exit(6)
		}
		matchedHosts.RunCommandOnHosts(matchedCommand.Command)
		for i := 0; i < len(matchedHosts); i++ {
			outputs = append(outputs, matchedHosts[i].Output)
		}
		printOutputWithCustomBanner(matchedCommand.Header, outputs)

	default:
		matchedCommand, partialCommands, err := matchCommand(fullCommand, commands)
		if err != nil {
			fmt.Printf("\nCouldn't match any command using labels '%v'. \n", fullCommand)
			fmt.Printf("Please check the '%v' file for the list of available commands. \n\n", yamlCommands)
			fmt.Printf("For running one time commands, you can use :\n")
			fmt.Printf("ybssh --exec '%v'\n\n", fullCommand)
		}
		if len(partialCommands) > 0 {
			fmt.Printf("Matched the following command labels :\n\n")
			showCommands(partialCommands)
			fmt.Println()
		}
		matchedHosts.RunCommandOnHosts(matchedCommand.Command)
	}
}
