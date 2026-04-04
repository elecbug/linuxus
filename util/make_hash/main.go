package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	var id, password string

	if len(os.Args) == 3 {
		args := os.Args[1:]
		id = strings.TrimSpace(args[0])
		password = strings.TrimSpace(args[1])
	} else if len(os.Args) == 1 {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("ID: ")
		id, _ = reader.ReadString('\n')
		id = strings.TrimSpace(id)

		fmt.Print("Password: ")
		password, _ = reader.ReadString('\n')
		password = strings.TrimSpace(password)
	} else {
		fmt.Println("Usage: go run main.go [ID password]")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Hash generation failed:", err)
		os.Exit(1)
	}

	fmt.Printf("%s:%s\n", id, string(hash))
}
