package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Configs pre-defined structure
type Configs struct {
	HostsFolder           string
	HostsFile             string
	CommandsFolder        string
	AuthType              int
	SummaryDetails        string
	SSHDefaultTimeout     int
	CommandDefaultTimeout int
}

// Config global instance containing the configuration provided in the config.yaml file
var Config Configs

func decrypt(keyFile string, securemess string) (decodedmess string, err error) {
	key := readFile(keyFile)
	cipherText, err := base64.StdEncoding.DecodeString(securemess)
	if err != nil {
		fmt.Printf("error: Password '%v' cannot be decrypted\n", securemess)
		os.Exit(1)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(cipherText) < aes.BlockSize {
		err = errors.New("Ciphertext block size is too short")
		return
	}

	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	decodedmess = string(cipherText)
	return
}

func readFile(path string) []byte {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	return content
}

func readConfigFile(fileName string) Configs {
	var config Configs
	var viperRuntime = viper.New()
	viperRuntime.SetConfigFile(fileName) // name of config file
	viperRuntime.SetConfigType("yaml")
	viperRuntime.AddConfigPath("/etc/gorun/")   // path to look for the config file in
	viperRuntime.AddConfigPath("$HOME/.gorun/") // call multiple times to add many search paths
	viperRuntime.AddConfigPath(".")             // optionally look for config in the working directory
	if err := viperRuntime.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Printf("Error finding yaml file: %s\n", err)
		}
		fmt.Printf("Error reading yaml file: %s\n", err)
		os.Exit(1)
	}
	err := viperRuntime.UnmarshalExact(&config)
	if err != nil {
		fmt.Printf("Unable to decode into struct, %v", err)
		os.Exit(1)
	}

	return config
}

func readAllCommandsFilesInFolder(folder string) ([]Command, error) {
	var allCommands []Command
	var files []string

	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".yaml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		err := errors.New(fmt.Sprintln("error: no yaml file found in folder ", folder))
		return nil, err
	}
	for _, file := range files {
		commands, err := readCommandsYamlFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		allCommands = append(allCommands, commands...)
	}

	return allCommands, nil
}

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
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
	}
	return myStruct, nil
}

func readAllHostsFilesInFolder(folder string, file string) (Nodes, error) {

	var allNodes Nodes
	var files []string
	if file == "" || file == "*" || file == "*.yaml" {
		err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
			if filepath.Ext(path) == ".yaml" {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			err := errors.New(fmt.Sprintln("error: no yaml file found in folder ", folder))
			return nil, err
		}
	} else {
		filePath := fmt.Sprintf("%v/%v", folder, file)
		files = append(files, filePath)
	}

	for _, file := range files {
		nodes, err := readHostsYamlFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		allNodes = append(allNodes, nodes...)
	}

	return allNodes, nil
}

func readHostsYamlFile(fileName string) (Nodes, error) {

	var myStruct []SSH
	var node Node
	var nodes Nodes
	var viperRuntime = viper.New()
	var defaults SSHDefaults

	viperRuntime.SetConfigName(fileName) // name of config file
	viperRuntime.SetConfigType("yaml")
	viperRuntime.AddConfigPath("/etc/gorun/")   // path to look for the config file in
	viperRuntime.AddConfigPath("$HOME/.gorun/") // call multiple times to add many search paths
	viperRuntime.AddConfigPath(".")             // optionally look for config in the working directory
	if err := viperRuntime.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Printf("Error finding YAML file: %s\n", err)
		}
		fmt.Printf("Error reading YAML file: %s\n", err)
		return nodes, err
	}

	err := viperRuntime.UnmarshalKey("defaults", &defaults)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
	}
	defaults.Password, err = decrypt(KeyFile, defaults.Password)
	if err != nil {
		fmt.Println(err)
	}

	err = viperRuntime.UnmarshalKey("nodes", &myStruct)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
	}
	for i := 0; i < len(myStruct); i++ {
		node.Client = myStruct[i]
		node.Client.Defaults = defaults
		nodes = append(nodes, node)
	}

	return nodes, nil
}
