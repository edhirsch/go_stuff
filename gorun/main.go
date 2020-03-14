package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type cliArgs struct {
	hostPattern  string
	primaryLabel string
	extraLabels  string
	extraArgs    string
}

func readStdinPipe() string {

	var pipe string
	info, err := os.Stdin.Stat()
	if err != nil {
		return ""
	}

	if info.Mode()&os.ModeCharDevice != 0 || info.Size() <= 0 {
		return ""
	}

	reader := bufio.NewReader(os.Stdin)
	var output []rune

	for {
		input, _, err := reader.ReadRune()
		if err != nil && err == io.EOF {
			break
		}
		output = append(output, input)
	}

	for _, rune := range output {
		pipe = pipe + fmt.Sprintf("%c", rune)
	}

	return pipe
}

func getArgs() (cliArgs, error) {
	var cli cliArgs
	args := os.Args[1:]
	if len(args) < 2 {
		return cli, errors.New("error: insufficient arguments")
	}
	cli.hostPattern = args[0]
	cli.primaryLabel = args[1]
	argsString := strings.Join(args[2:], " ")
	argsSplit := regexp.MustCompile(" -- ").Split(argsString, -1)
	if len(argsSplit) == 0 {
		cli.extraLabels = argsString
	} else if len(argsSplit) == 1 {
		cli.extraLabels = argsSplit[0]
	} else {
		cli.extraLabels = argsSplit[0]
		cli.extraArgs = argsSplit[len(argsSplit)-1]
	}

	return cli, nil
}

func listCommands(commands []Command) {
	var lines []string
	lines = append(lines, "LABELS\tCOMMAND\tDESCRIPTION")
	for _, command := range commands {
		line := fmt.Sprintf("%v\t%.35v\t%v", command.Name, command.Command, command.Description)
		lines = append(lines, line)
	}
	printTabbedTable(lines)
}

func listMatchedHosts(nodes Nodes) {
	var lines []string
	lines = append(lines, "NODES")
	for _, node := range nodes {
		line := fmt.Sprintf("%v@%v:%v", node.Client.User, node.Client.Server, node.Client.Port)
		lines = append(lines, line)
	}
	printTabbedTable(lines)
}

func showHelp() {
	help := `Usage :
	gorun <hosts> <command>
	gorun <hosts> --commands
	gorun <hosts> --exec <command>
	gorun <hosts> --list
	gorun <hosts> --play <script>
	`
	fmt.Println(help)
}

func main() {
	Config = readConfigFile("config.yaml")
	KeyFile = os.Getenv("HOME") + "/.gorun/.config"
	pipe := readStdinPipe()

	var fullCommand string

	hosts, err := readAllHostsFilesInFolder(Config.HostsFolder, Config.HostsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	for i := 0; i < len(hosts); i++ {
		hosts[i].Client.initHosts()
	}

	commands, err := readAllCommandsFilesInFolder(Config.CommandsFolder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	initCommands(commands)

	cli, err := getArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		showHelp()
		return
	}

	fullCommand = cli.primaryLabel + " " + cli.extraLabels

	matchedHosts, err := matchHost(cli.hostPattern, hosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	switch cli.primaryLabel {
	case "--commands":
		listCommands(commands)
		fmt.Println()
		break

	case "--exec":
		var execCommand Command
		if len(pipe) > 0 {
			if cli.extraLabels != "" {
				execCommand.Command = fmt.Sprintf("%v '%v'", cli.extraLabels, pipe)
			} else {
				execCommand.Command = pipe
			}
			execCommand.Name = "pipe command(s)"
		} else {
			execCommand.Command = cli.extraLabels
			execCommand.Args = cli.extraArgs
			execCommand.Name = cli.extraLabels
		}
		runCommandOnHosts(execCommand, matchedHosts)

	case "--list":
		listMatchedHosts(matchedHosts)
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
			listCommands(partialCommands)
			fmt.Println()
			return
		}
		matchedCommand.Args = cli.extraArgs
		runCommandOnHosts(matchedCommand, matchedHosts)
	}
}
