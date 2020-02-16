package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

// Flow yaml struct
type Flow struct {
	sequential bool
	parallel   bool
	selection  string
	iteration  string
}

// Output yaml struct
type Output struct {
	raw      string
	filter   string
	filtered string
	variable string
}

// Command yaml struct
type Command struct {
	Name       string   `yaml:"name"`
	Command    string   `yaml:"command"`
	Parameters []string `yaml:"parameters"`
	Depends    string   `yaml:"depends"`
	Output     Output   `yaml:"output,inline"`
	Flow       Flow     `yaml:"flow,inline"`
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
