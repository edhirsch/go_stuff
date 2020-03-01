package main

import (
	"fmt"
	"io/ioutil"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

func readCommandsYamlFile(fileName string) ([]Command, error) {
	var myStruct []Command
	var viperRuntime = viper.New()
	viperRuntime.SetConfigName(fileName) // name of config file (without extension)
	viperRuntime.SetConfigType("yaml")
	viperRuntime.AddConfigPath("/etc/gorun/")   // path to look for the config file in
	viperRuntime.AddConfigPath("$HOME/.gorun/") // call multiple times to add many search paths
	viperRuntime.AddConfigPath(".")             // optionally look for config in the working directory
	if err := viperRuntime.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Printf("Error finding YAML file: %s\n", err)
		}
		fmt.Printf("Error reading YAML file: %s\n", err)
		return nil, err
	}
	err := viperRuntime.UnmarshalKey("commands", &myStruct)
	// err := yaml.Unmarshal(viperRuntime.UnmarshalKey(), &commands)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
	}
	return myStruct, nil
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
