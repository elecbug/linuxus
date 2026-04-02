package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	var studentID, password string

	if len(os.Args) == 3 {
		args := os.Args[1:]
		studentID = strings.TrimSpace(args[0])
		password = strings.TrimSpace(args[1])
	} else if len(os.Args) == 1 {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Student ID: ")
		studentID, _ = reader.ReadString('\n')
		studentID = strings.TrimSpace(studentID)

		fmt.Print("Password: ")
		password, _ = reader.ReadString('\n')
		password = strings.TrimSpace(password)
	} else {
		fmt.Println("Usage: go run main.go [studentID password]")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Hash generation failed:", err)
		os.Exit(1)
	}

	fmt.Printf("%s:%s\n", studentID, string(hash))
}
