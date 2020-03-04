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
)

// Constants
const (
	CertPassword           = 1
	CertPublicKeyFile      = 2
	DefaultTimeout         = 3 // second
	Debug             bool = false
)

// KeyFile ; generate via gokey
var KeyFile string

// AuthType ; CertPassword || CertPublicKeyFile
var AuthType int

// Command pre-defined struct
// ------------------------------------
type Command struct {
	Name        string `yaml:"name"`
	Command     string `yaml:"command"`
	Args        string `yaml:"args"`
	Description string `yaml:"description"`
	Header      string `yaml:"header"`
	Output      string
	ReturnCode  int
}

// Node pre-defined struct
type Node struct {
	Client     SSH
	Output     string
	ReturnCode int
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

func getArgs() (string, string, string, string, error) {
	var hostPattern, commandLabel, otherCommandLabels, commandExtraArgs string
	args := os.Args[1:]
	if len(args) < 2 {
		return "", "", "", "", errors.New("error: insufficient arguments")
	}
	hostPattern = args[0]
	commandLabel = args[1]
	argsString := strings.Join(args[2:], " ")
	argsSplit := regexp.MustCompile("--").Split(argsString, -1)
	if len(argsSplit) == 0 {
		otherCommandLabels = argsString
	} else if len(argsSplit) == 1 {
		otherCommandLabels = argsSplit[0]
	} else {
		otherCommandLabels = argsSplit[0]
		commandExtraArgs = argsSplit[len(argsSplit)-1]
	}

	return hostPattern, commandLabel, otherCommandLabels, commandExtraArgs, nil
}

func getDefaultBanner(command string, duration string, rc string, sshClient SSH) string {
	var banner string
	x := strings.Repeat("-",
		utf8.RuneCountInString(sshClient.Server)+
			utf8.RuneCountInString(sshClient.Port)+
			utf8.RuneCountInString(command)+utf8.RuneCountInString(duration)+
			utf8.RuneCountInString(rc)+
			37)
	banner = banner + fmt.Sprintf("%v\n", x)
	banner = banner + fmt.Sprintf("| %v:%v | command: %v | duration: %v | rc: %v |\n",
		sshClient.Server, sshClient.Port, command, duration, rc)
	banner = banner + fmt.Sprintf("%v\n", x)

	return banner
}

func getSummaryBanner(command string, duration string, passed string, failed string, total string) string {
	var banner string
	x := strings.Repeat("-",
		utf8.RuneCountInString(command)+
			utf8.RuneCountInString(duration)+
			utf8.RuneCountInString(passed)+
			utf8.RuneCountInString(failed)+
			utf8.RuneCountInString(total)+
			68)
	banner = banner + fmt.Sprintf("%v\n", x)
	banner = banner + fmt.Sprintf("| summary | command: %v | duration: %v | passed: %v | failed: %v | total: %v |\n",
		command, duration, passed, failed, total)
	banner = banner + fmt.Sprintf("%v\n", x)

	return banner
}

func runCommandOnHosts(command Command, sshClients Nodes) {
	var wg sync.WaitGroup
	tt1 := time.Now()
	for i := 0; i < len(sshClients); i++ {
		c := make(chan string)
		e := make(chan error)
		t1 := time.Now()
		wg.Add(1)
		runCommand := command.Command + " " + command.Args
		go runCommandParallel(runCommand, sshClients[i].Client, &wg, c, e)
		go func(sshClient *Node) {
			output := <-c
			err := <-e
			if err != nil {
				sshClient.ReturnCode = 1
			}
			tdiff := time.Now().Sub(t1)
			duration := fmt.Sprintf("%0.2vs", tdiff.Seconds())
			rc := fmt.Sprintf("%v", sshClient.ReturnCode)
			if command.Header == "" {
				banner := getDefaultBanner(runCommand, duration, rc, sshClient.Client)
				if sshClient.ReturnCode == 0 {
					sshClient.Output = Green(banner) + Default(output)
				} else {
					sshClient.Output = Red(banner) + Black(output)
				}
				fmt.Printf("%v\n\n", sshClient.Output)
			} else {
				sshClient.Output = output
			}
		}(&sshClients[i])
		time.Sleep(10 * time.Millisecond)
	}
	wg.Wait()
	tdiff := time.Now().Sub(tt1)
	totalDuration := fmt.Sprintf("%0.2vs", tdiff.Seconds())
	if command.Header != "" {
		outputs := getAllOutputs(sshClients)
		printOutputWithCustomBanner(command.Header, outputs)
	} else {
		printCommandSummary(sshClients, command.Name, totalDuration)
	}
}

func getAllOutputs(sshClients Nodes) []string {
	var outputs []string
	for i := 0; i < len(sshClients); i++ {
		outputs = append(outputs, sshClients[i].Output)
	}
	return outputs
}

func printCommandSummary(sshClients Nodes, command string, duration string) {
	var passed, failed int
	var summary []string

	for i := 0; i < len(sshClients); i++ {
		serverAndPort := fmt.Sprintf("%v:%v", sshClients[i].Client.Server, sshClients[i].Client.Port)
		if sshClients[i].ReturnCode > 0 {
			failed++
			if Config.SummaryDetails == "failed-only" || Config.SummaryDetails == "all" {
				summary = append(summary, fmt.Sprintf("%v -> %v", serverAndPort, Red("FAILED")))
			}
		} else {
			passed++
			if Config.SummaryDetails == "passed-only" || Config.SummaryDetails == "all" {
				summary = append(summary, fmt.Sprintf("%v -> %v", serverAndPort, Green("PASSED")))
			}
		}
	}
	total := len(sshClients)
	banner := getSummaryBanner(command, duration, fmt.Sprintf("%v", passed), fmt.Sprintf("%v", failed), fmt.Sprintf("%v", total))

	if total == passed {
		fmt.Printf("%v", Green(banner))
	} else if total == failed {
		fmt.Printf("%v", Red(banner))
	} else {
		fmt.Printf("%v", Yellow(banner))
	}

	for i := 0; i < len(summary); i++ {
		fmt.Printf("%v\n", summary[i])
	}
}

func runCommandParallel(command string, sshClient SSH, wg *sync.WaitGroup, c chan string, e chan error) {
	defer wg.Done()
	err := sshClient.Connect(CertPassword)
	if err != nil {
		c <- fmt.Sprintln(err)
		e <- err
		return
	}
	sshClient.RefreshSession()
	commandOutput, err := sshClient.RunCommand(command)

	c <- commandOutput
	e <- err
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
	primaryCommandLabels := strings.Fields(commandString)
	sort.Strings(primaryCommandLabels)
	for i := 0; i < len(commandList); i++ {
		commandListLabels := strings.Fields(commandList[i].Name)
		sort.Strings(commandListLabels)
		if reflect.DeepEqual(commandListLabels, primaryCommandLabels) == true {
			foundCommand = commandList[i]
		} else {
			if matchArrayInArray(primaryCommandLabels, commandListLabels) == true {
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
	gorun <hosts> <command labels>
	gorun <hosts> commands
	gorun <hosts> exec <command>
	gorun <hosts> play <script>
	`
	fmt.Println(help)
}

func main() {

	Config = readConfigFile("config.yaml")
	KeyFile = os.Getenv("HOME") + "/.gorun/.config"
	if Config.AuthType == "password" {
		AuthType = CertPassword
	}
	var fullCommand string

	hosts, err := readHostsYamlFile(Config.HostsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	for i := 0; i < len(hosts); i++ {
		hosts[i].Client.init()
	}
	commands, err := readAllCommandsFilesInFolder(Config.CommandsFolder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	hostsPattern, primaryCommandLabel, otherCommandLabels, commandArgs, err := getArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		showHelp()
		return
	}
	fullCommand = primaryCommandLabel + " " + otherCommandLabels

	matchedHosts, err := matchHost(hostsPattern, hosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	switch primaryCommandLabel {
	case "exec":
		var execCommand Command
		execCommand.Command = otherCommandLabels
		execCommand.Args = commandArgs
		execCommand.Name = otherCommandLabels
		runCommandOnHosts(execCommand, matchedHosts)

	case "commands":
		showCommands(commands)
		fmt.Println()
		break

	default:
		matchedCommand, partialCommands, err := matchCommand(fullCommand, commands)
		if err != nil {
			fmt.Printf("\nCouldn't match any command using labels '%v'. \n", fullCommand)
			fmt.Printf("Please check the commands files in '%v' for the list of available commands. \n\n", Config.CommandsFolder)
			fmt.Printf("For running one time commands, you can use :\n")
			fmt.Printf("gorun --exec '%v'\n\n", fullCommand)
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}
		if len(partialCommands) > 0 {
			fmt.Printf("Matched the following command labels :\n\n")
			showCommands(partialCommands)
			fmt.Println()
			return
		}
		matchedCommand.Args = commandArgs
		runCommandOnHosts(matchedCommand, matchedHosts)
	}
}
