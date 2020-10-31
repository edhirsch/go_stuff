package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type cliArgs struct {
	scriptName  string
	hostPattern string
	command     string
	args        string
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
	scriptPathSlice := strings.Split(os.Args[0], "/")
	cli.scriptName = scriptPathSlice[len(scriptPathSlice)-1]
	args := os.Args[1:]
	if len(args) < 2 {
		return cli, errors.New("error: insufficient arguments")
	}
	cli.hostPattern = args[0]
	cli.command = args[1]
	cli.args = ""
	if len(args) > 2 {
		cli.args = strings.Join(args[2:], " ")
	}

	return cli, nil
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

func showHelp(scriptName string) {
	help := `Usage :
	scriptName <hosts> <command>
	scriptName <hosts> --list
	`
	help = strings.ReplaceAll(help, "scriptName", scriptName)
	fmt.Println(help)
}

func main() {
	Config = readConfigFile("config.yaml")
	KeyFile = os.Getenv("HOME") + "/.gorun/.config"
	pipe := readStdinPipe()

	hosts, err := readAllHostsFilesInFolder(Config.HostsFolder, Config.HostsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	for i := 0; i < len(hosts); i++ {
		hosts[i].Client.initHosts()
	}

	cli, err := getArgs()
	if err != nil {
		showHelp(cli.scriptName)
		return
	}

	matchedHosts, err := matchHost(cli.hostPattern, hosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	switch cli.command {

	case "--list":
		listMatchedHosts(matchedHosts)
		break

	default:
		var execCommand Command
		execCommand.Command = cli.command
		execCommand.Args = cli.args
		execCommand.Name = cli.command
		execCommand.Pipe = pipe
		runCommandOnHosts(execCommand, matchedHosts)
	}
}
