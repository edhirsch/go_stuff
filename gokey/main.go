package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	rnd "math/rand"

	"golang.org/x/crypto/ssh/terminal"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func readKeyFromFile(path string) []byte {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	return content
}

func encrypt(keyFile string, message string) (encmess string, err error) {
	key := readKeyFromFile(keyFile)
	plainText := []byte(message)

	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	//IV needs to be unique, but doesn't have to be secure.
	//It's common to put it at the beginning of the ciphertext.
	cipherText := make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)

	//returns to base64 encoded string
	encmess = base64.URLEncoding.EncodeToString(cipherText)
	return
}

func decrypt(keyFile string, securemess string) (decodedmess string, err error) {
	key := readKeyFromFile(keyFile)
	cipherText, err := base64.URLEncoding.DecodeString(securemess)
	if err != nil {
		return
	}

	block, err := aes.NewCipher(key)
	if err != nil {
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

func generateKey() {
	folderName := os.Getenv("HOME") + "/.gorun/"
	fileName := ".config"
	fullPath := folderName + fileName
	hostname, err := os.Hostname()
	rnd.Seed(time.Now().UnixNano())
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	key := []byte(hostname)
	for i := 0; i < 32-len(hostname); i++ {
		key = append(key, byte(letterRunes[rnd.Intn(len(letterRunes))]))
	}

	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		os.Mkdir(folderName, 0755)
	}
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	os.OpenFile(fullPath, os.O_RDONLY|os.O_CREATE, 0644)
	err = ioutil.WriteFile(fullPath, key, 0644)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}

func getArgs() (string, error) {

	if len(os.Args) < 2 {
		return "", errors.New("error: insufficient arguments")
	}
	return os.Args[1], nil
}

func inputString() (string, error) {
	fmt.Printf("Input a string to encrypt: \n")
	text, err := terminal.ReadPassword(0)
	if err != nil {
		return "", err
	}

	return string(text), nil
}

func main() {
	folderName := os.Getenv("HOME") + "/.gorun/"
	fileName := ".config"
	fullPath := folderName + fileName

	command, err := getArgs()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	switch command {
	case "generate":
		generateKey()
	case "encrypt":
		text, err := inputString()
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		encrypted, err := encrypt(fullPath, text)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		fmt.Println(encrypted)
	case "decrypt":
		text, err := inputString()
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		decrypted, err := decrypt(fullPath, text)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		fmt.Println(decrypted)
	}
}
