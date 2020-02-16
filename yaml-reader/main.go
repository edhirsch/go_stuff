package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

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

// Command yaml struct
type Command struct {
	Name       string   `yaml:"name"`
	Command    string   `yaml:"command"`
	Parameters []string `yaml:"parameters"`
	Depends    string   `yaml:"depends"`
	Output     Output   `yaml:",flow"`
	Flow       Flow     `yaml:",flow"`
}

// Host yaml struct
type Host struct {
	Server   string `yaml:"server"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func main() {
	var yamlDefaults = "default.yaml"
	cmds, err := ReadYamlFile(yamlDefaults)
	if err != nil {
		os.Exit(1)
	}
	fmt.Printf("%v\n", cmds)

}

// ReadYamlFile function
func ReadYamlFile(fileName string) ([]Command, error) {

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
