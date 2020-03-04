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

	"github.com/spf13/viper"
)

// Configs pre-defined structure
type Configs struct {
	HostsFile      string
	CommandsFile   string
	AuthType       string
	SummaryDetails string
}

// Config globals
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

	//IV needs to be unique, but doesn't have to be secure.
	//It's common to put it at the beginning of the ciphertext.
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	// XORKeyStream can work in-place if the two arguments are the same.
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
	viperRuntime.SetConfigFile(fileName) // name of config file (without extension)
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

func readHostsYamlFile(fileName string) (Nodes, error) {

	var myStruct []SSH
	var node Node
	var nodes Nodes
	// var defaults Defaults
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
		return nodes, err
	}
	err := viperRuntime.UnmarshalKey("nodes", &myStruct)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
	}
	for i := 0; i < len(myStruct); i++ {
		node.Client = myStruct[i]
		nodes = append(nodes, node)
	}
	err = viperRuntime.UnmarshalKey("defaults", &DefaultConfig)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
	}
	DefaultConfig.Password, err = decrypt(KeyFile, DefaultConfig.Password)
	if err != nil {
		fmt.Println(err)
	}

	return nodes, nil
}
