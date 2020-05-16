package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// ReadScriptYamlFile function
func ReadScriptYamlFile(fileName string) ([]Script, error) {

	var yamlConfig []Script
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

	if Debug {
		fmt.Println(yamlConfig)
	}

	return yamlConfig, nil
}

// ReadCommandsYamlFile function
func ReadCommandsYamlFile(fileName string) ([]Command, error) {

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

	if Debug {
		fmt.Println(yamlConfig)
	}

	return yamlConfig, nil
}

// ReadHostsYamlFile function
func ReadHostsYamlFile(fileName string) (Execution, error) {

	var yamlConfig Execution
	yamlFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return yamlConfig, err
	}

	err = yaml.Unmarshal([]byte(yamlFile), &yamlConfig.MultiSSH)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
		return yamlConfig, err
	}

	if Debug {
		fmt.Println(yamlConfig.MultiSSH)
	}

	return yamlConfig, nil
}
