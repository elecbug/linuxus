package main

import (
	"log"

	"github.com/elecbug/linuxus/src/internal/manager"
)

func main() {
	if err := manager.Run(); err != nil {
		log.Fatal(err)
	}
}
