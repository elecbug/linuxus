package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Student ID: ")
	studentID, _ := reader.ReadString('\n')
	studentID = strings.TrimSpace(studentID)

	fmt.Print("Password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Hash generation failed:", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("%s:%s\n", studentID, string(hash))
}
