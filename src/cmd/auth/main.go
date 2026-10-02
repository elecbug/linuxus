package main

import (
	"log"

	"github.com/elecbug/linuxus/src/internal/auth"
)

func main() {
	if err := auth.Run(); err != nil {
		log.Fatal(err)
	}
}
