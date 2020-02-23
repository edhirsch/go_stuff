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
	Output      string
	ReturnCode  int
}

// MultiSSH pre-defined struct
type MultiSSH []SSH

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

// PrintCommandOutput function
func (sshClient *SSH) PrintCommandOutput(output string, command string) {
	if Banner == true {
		x := strings.Repeat("-", utf8.RuneCountInString(sshClient.Server)+utf8.RuneCountInString(sshClient.Port)+
			utf8.RuneCountInString(command)+7)
		fmt.Printf("%v\n", x)
		fmt.Printf("| %v:%v; %v |\n", sshClient.Server, sshClient.Port, command)
		fmt.Printf("%v\n", x)
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fmt.Printf("%v\n", line)
	}
}

// RunCommandOnHosts function
func (sshClients MultiSSH) RunCommandOnHosts(command string) {
	var wg sync.WaitGroup
	for i := 0; i < len(sshClients); i++ {
		sshClients[i].Connect(CertPassword)
		wg.Add(1)
		go sshClients[i].RunCmdParallel(command, &wg)
	}
	wg.Wait()
}

// RunCmdParallel function
func (sshClient SSH) RunCmdParallel(command string, wg *sync.WaitGroup) {
	defer wg.Done()
	sshClient.RefreshSession()
	commandOutput := sshClient.RunCmd(command)
	sshClient.PrintCommandOutput(commandOutput, command)
	sshClient.Close()
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
	case "commands":
		showCommands(commands)
		fmt.Println()
		break
	case "nodes":
		Banner = false
		matchedCommand, _, err := matchCommand("list nodes", commands)
		if err != nil {
			os.Exit(6)
		}
		matchedHosts.RunCommandOnHosts(matchedCommand.Command)
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
