package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

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
func ReadHostsYamlFile(fileName string) (Nodes, error) {

	var yamlNodes []SSH
	var node Node
	var nodes Nodes

	yamlFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return nodes, err
	}

	err = yaml.Unmarshal([]byte(yamlFile), &yamlNodes)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
		return nodes, err
	}
	for i := 0; i < len(yamlNodes); i++ {
		node.Client = yamlNodes[i]
		nodes = append(nodes, node)
	}

	if Debug {
		fmt.Println(nodes)
	}

	return nodes, nil
}
