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

// Variables global
var Variables []Variable

// Command yaml pre-defined struct
// ------------------------------------
type Command struct {
	Name        string   `yaml:"name"`
	Command     string   `yaml:"command"`
	Description string   `yaml:"description"`
	Filters     []Filter `yaml:",flow"`
	Output      string
	ReturnCode  int
}

// Execution pre-defined structure
type Execution struct {
	MultiSSH []SSH
	Command
	Scripts []Script
}

// Filter yaml struct
type Filter struct {
	RegEx string
	Save  string
}

// Script yaml pre-defined struct
// ------------------------------------
type Script struct {
	ID      int    `yaml:"id"`
	Command string `yaml:"command"`
	Loop    Loop   `yaml:",flow"`
	Next    []Next `yaml:",flow"`
}

// Next yaml pre-defined struct
type Next struct {
	Condition string
	Run       int
}

// Loop yaml struct
type Loop struct {
	Repeat int
	Break  string
}

// // Output yaml struct
// type Output struct {
// 	Stdout string
// 	Stderr string
// }

// Variable struct
type Variable struct {
	Name  string
	Value string
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
func (sshClients Execution) RunCommandOnHosts() {
	var wg sync.WaitGroup
	for i := 0; i < len(sshClients.MultiSSH); i++ {
		sshClients.MultiSSH[i].Connect(CertPassword)
		wg.Add(1)
		go sshClients.MultiSSH[i].RunCmdParallel(sshClients.Command.Command, &wg)
	}
	wg.Wait()
}

// RunCmdParallel function
func (sshClient SSH) RunCmdParallel(command string, wg *sync.WaitGroup) {
	defer wg.Done()
	var setVar Variable
	sshClient.RefreshSession()
	commandOutput := sshClient.RunCmd(command)
	// setVar.Name = sshClient.Server + ":" + sshClient.Port + ".'" + command + "'.output"
	// setVar.Value = commandOutput
	Variables = append(Variables, setVar)

	sshClient.PrintCommandOutput(commandOutput, command)
	sshClient.Close()
}

// RunScriptOnHosts function
func (sshClients Execution) RunScriptOnHosts() {
	var wg sync.WaitGroup
	for i := 0; i < len(sshClients.MultiSSH); i++ {
		sshClients.MultiSSH[i].Connect(CertPassword)
		wg.Add(1)
		go sshClients.MultiSSH[i].findAndRunScript(sshClients.Scripts, &wg)
	}
	wg.Wait()

	for i := 0; i < len(sshClients.MultiSSH); i++ {
		sshClients.MultiSSH[i].Close()
	}
}

// FindAndRunScript function
func (sshClient *SSH) findAndRunScript(scripts []Script, wg *sync.WaitGroup) {
	defer wg.Done()
	var id []int
	id = append(id, 1)
	var script Script
	// itterate and run all commands in the command ID splice, starting from command ID 1
	for len(id) > 0 {
		for i := 0; i < len(id); i++ {
			for j := 0; j < len(scripts); j++ {
				if scripts[j].ID == id[i] {
					script = scripts[j]
					break
				}
			}
			// remove the already selected command ID from the queue
			id = id[1:]
			// run the command once and again for the value of 'repeat' or the condition is satisfied
			for r := 0; r <= script.Loop.Repeat; r++ {
				sshClient.RefreshSession()
				cmdOutput := sshClient.RunCmd(script.Command)
				sshClient.PrintCommandOutput(cmdOutput, script.Command)
			}
			// add all next command IDs to the commands queue if the condition passes
			if len(script.Next) > 0 {
				for k := 0; k < len(script.Next); k++ {
					id = append(id, script.Next[k].Run)
				}
			}
		}

	}
}

func matchHost(hostString string, hostsList Execution) (Execution, error) {
	var foundHosts Execution
	for i := 0; i < len(hostsList.MultiSSH); i++ {
		matched, err := regexp.MatchString(hostString, hostsList.MultiSSH[i].Server)
		if err == nil {
			if matched {
				foundHosts.MultiSSH = append(foundHosts.MultiSSH, hostsList.MultiSSH[i])
			}
		}
	}
	if len(foundHosts.MultiSSH) == 0 {
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

func showHelp() {
	help := `Usage :
	ybssh <hosts> <command labels>
	ybssh <hosts> --exec <command>
	ybssh <hosts> --script <script.yaml>
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
		matchedHosts.Command.Command = argsArg
		matchedHosts.RunCommandOnHosts()
	case "--apply":
		yamlScript := argsArg
		script, err := ReadScriptYamlFile(yamlScript)
		if err != nil {
			os.Exit(5)
		}
		matchedHosts.Scripts = script
		matchedHosts.RunScriptOnHosts()
		for v := 0; v < len(Variables); v++ {
			fmt.Printf("%v: %v", Variables[v].Name, Variables[v].Value)
		}
	case "--list":
		Banner = false
		matchedCommand, _, err := matchCommand("list nodes", commands)
		if err != nil {
			os.Exit(6)
		}
		matchedHosts.Command = matchedCommand
		matchedHosts.RunCommandOnHosts()
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
			fmt.Printf("Matched the following command labels :\n\n")
			var lines []string
			lines = append(lines, "LABELS\tCOMMAND\tDESCRIPTION")
			for i := 0; i < len(partialCommands); i++ {
				line := fmt.Sprintf("%v\t%v\t%v", partialCommands[i].Name, partialCommands[i].Command, partialCommands[i].Description)
				lines = append(lines, line)
			}
			printTabbedTable(lines)
			fmt.Println()
			break
		}
		matchedHosts.Command.Command = matchedCommand.Command
		matchedHosts.RunCommandOnHosts()
	}
}
