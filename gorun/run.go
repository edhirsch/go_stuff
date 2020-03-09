package main

import (
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

// Command pre-defined struct
// ------------------------------------
type Command struct {
	Name        string `yaml:"name"`
	Command     string `yaml:"command"`
	Args        string `yaml:"args"`
	Description string `yaml:"description"`
	Header      string `yaml:"header"`
	Timeout     int    `yaml:"timeout"`
	Output      string
	ReturnCode  int
}

func initCommands(commands []Command) {
	for _, command := range commands {
		if command.Timeout == 0 {
			command.Timeout = Config.CommandDefaultTimeout
		}
	}
}

func printTabbedTable(lines []string) {
	writer := tabwriter.NewWriter(os.Stdout, 20, 8, 1, '\t', tabwriter.AlignRight)
	for i := 0; i < len(lines); i++ {
		fmt.Fprintln(writer, lines[i])
	}
	writer.Flush()
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
		runCommand := command.Command
		if command.Args != "" {
			runCommand = runCommand + " " + command.Args
		}

		go runCommandParallel(runCommand, command.Timeout, sshClients[i].Client, &wg, c, e)
		go func(sshClient *Node) {
			defer wg.Done()
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

func runCommandParallel(command string, timeout int, sshClient SSH, wg *sync.WaitGroup, c chan string, e chan error) {
	if timeout > 0 {
		command = fmt.Sprintf("timeout --kill-after=%v %v bash -c '%v'", timeout, timeout, command)
	}
	err := sshClient.Connect(Config.AuthType)
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

func printOutputWithCustomBanner(banner string, output []string) {
	var lines []string
	banner = fmt.Sprintf(banner)
	lines = append(lines, banner)
	lines = append(lines, output...)
	printTabbedTable(lines)
}

func matchHost(hostPatterns string, hostsList Nodes) (Nodes, error) {
	var foundHosts Nodes
	for _, pattern := range strings.Split(hostPatterns, ",") {
		for _, host := range hostsList {
			matched, err := regexp.MatchString(pattern, host.Client.Server)
			if err == nil {
				if matched {
					exists := false
					for _, existinghost := range foundHosts {
						if host == existinghost {
							exists = true
							break
						}
					}
					if exists == false {
						foundHosts = append(foundHosts, host)
					}
				}
			}
		}
		if len(foundHosts) == 0 {
			return foundHosts, fmt.Errorf("error: couldn't match any hosts using the provided pattern '%v'", hostPatterns)
		}
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
