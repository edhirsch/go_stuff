package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

func decryptPasswords(textBytes []byte, keyFile string) []byte {
	text := string(textBytes)
	var re = regexp.MustCompile(`(?m)password: (["|'].*["|'])$`)
	var matched []string
	for _, match := range re.FindAllStringSubmatch(text, -1) {
		if match[1] != "" {
			matched = appendIfMissing(matched, match[1])
		}
	}
	decryptedText := text
	for _, match := range matched {
		encryptedPassword := trimFirstLastChars(match, 1, len(match)-1)
		textPassword, err := decrypt(keyFile, encryptedPassword)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		decryptedText = strings.ReplaceAll(decryptedText, encryptedPassword, textPassword)
	}
	return []byte(decryptedText)
}

func decrypt(keyFile string, securemess string) (decodedmess string, err error) {
	key := readFile(keyFile)
	cipherText, err := base64.StdEncoding.DecodeString(securemess)
	if err != nil {
		fmt.Println(err)
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

func trimFirstLastChars(str string, firstIndex int, lastIndex int) string {
	r := []rune(str)
	return string(r[firstIndex:lastIndex])
}

func appendIfMissing(strSlice []string, str string) []string {
	for _, ele := range strSlice {
		if ele == str {
			return strSlice
		}
	}
	return append(strSlice, str)
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

// // ReadCommandsYamlFile function
// func ReadCommandsYamlFile(fileName string) ([]Command, error) {

// 	var yamlConfig []Command
// 	yamlFile, err := ioutil.ReadFile(fileName)
// 	if err != nil {
// 		fmt.Printf("Error reading YAML file: %s\n", err)
// 		return yamlConfig, err
// 	}

// 	err = yaml.Unmarshal([]byte(yamlFile), &yamlConfig)
// 	if err != nil {
// 		fmt.Printf("Error parsing YAML file: %s\n", err)
// 		return yamlConfig, err
// 	}

// 	if Debug {
// 		fmt.Println(yamlConfig)
// 	}

// 	return yamlConfig, nil
// }

// // ReadHostsYamlFile function
// func ReadHostsYamlFile(fileName string, keyFile string) (Nodes, error) {

// 	var yamlNodes []SSH
// 	var node Node
// 	var nodes Nodes

// 	yamlFile, err := ioutil.ReadFile(fileName)
// 	if err != nil {
// 		fmt.Printf("Error reading YAML file: %s\n", err)
// 		return nodes, err
// 	}
// 	yamlFileDecrypted := decryptPasswords(yamlFile, keyFile)
// 	err = yaml.Unmarshal([]byte(yamlFileDecrypted), &yamlNodes)
// 	if err != nil {
// 		fmt.Printf("Error parsing YAML file: %s\n", err)
// 		return nodes, err
// 	}
// 	for i := 0; i < len(yamlNodes); i++ {
// 		node.Client = yamlNodes[i]
// 		nodes = append(nodes, node)
// 	}

// 	if Debug {
// 		fmt.Println(nodes)
// 	}

// 	return nodes, nil
// }
