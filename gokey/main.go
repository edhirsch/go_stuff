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
	"regexp"
	"strings"
	"time"

	rnd "math/rand"

	"golang.org/x/crypto/ssh/terminal"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func readFile(path string) []byte {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	return content
}

func encrypt(keyFile string, message string) (encmess string, err error) {
	key := readFile(keyFile)
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
	encmess = base64.StdEncoding.EncodeToString(cipherText)
	return
}

func decrypt(keyFile string, securemess string) (decodedmess string, err error) {
	key := readFile(keyFile)
	cipherText, err := base64.StdEncoding.DecodeString(securemess)
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

func getArgs() (string, string, error) {

	if len(os.Args) <= 1 {
		return "", "", errors.New("error: insufficient arguments")
	} else if len(os.Args) == 2 {
		return os.Args[1], "", nil
	} else {
		return os.Args[1], os.Args[2], nil
	}
}

func inputString() (string, error) {
	fmt.Printf("Input a string to encrypt: \n")
	text, err := terminal.ReadPassword(0)
	if err != nil {
		return "", err
	}

	return string(text), nil
}

func encryptPasswordInFile(hostsFile string, keyFile string) {
	var re = regexp.MustCompile(`(?m)password: (["|'].*["|'])$`)
	var matched []string
	text := string(readFile(hostsFile))
	for _, match := range re.FindAllStringSubmatch(text, -1) {
		if match[1] != "" {
			matched = appendIfMissing(matched, match[1])
		}

	}
	encryptedText := text
	for _, match := range matched {
		textPassword := trimFirstLastChars(match, 1, len(match)-1)
		encryptedPassword, err := encrypt(keyFile, textPassword)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		fmt.Println(textPassword, encryptedPassword)
		encryptedText = strings.ReplaceAll(encryptedText, textPassword, encryptedPassword)
	}
	err := ioutil.WriteFile(hostsFile, []byte(encryptedText), 0644)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}

func decryptPasswordInFile(hostsFile string, keyFile string) {
	var re = regexp.MustCompile(`(?m)password: (["|'].*["|'])$`)
	var matched []string
	text := string(readFile(hostsFile))
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
		fmt.Println(textPassword, encryptedPassword)
		decryptedText = strings.ReplaceAll(decryptedText, encryptedPassword, textPassword)
	}
	err := ioutil.WriteFile(hostsFile, []byte(decryptedText), 0644)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
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

func main() {
	folderName := os.Getenv("HOME") + "/.gorun/"
	fileName := ".config"
	fullPath := folderName + fileName

	command, arg, err := getArgs()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	switch command {
	case "generate":
		generateKey()
	case "encrypt":
		if arg == "" {
			text, err := inputString()
			if err != nil {
				fmt.Printf("%v\n", err)
			}
			encrypted, err := encrypt(fullPath, text)
			if err != nil {
				fmt.Printf("%v\n", err)
			}
			fmt.Println(encrypted)
		} else {
			encryptPasswordInFile(arg, fullPath)
		}
	case "decrypt":
		if arg == "" {
			text, err := inputString()
			if err != nil {
				fmt.Printf("%v\n", err)
			}
			decrypted, err := decrypt(fullPath, text)
			if err != nil {
				fmt.Printf("%v\n", err)
			}
			fmt.Println(decrypted)
		} else {
			decryptPasswordInFile(arg, fullPath)
		}
	}
}
