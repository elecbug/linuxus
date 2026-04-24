package format

import (
	"fmt"
	"log"
)

type LogLevel string

const (
	HEADER_PREFIX  LogLevel = "[+]"
	RUN_PREFIX     LogLevel = "[>]"
	ERROR_PREFIX   LogLevel = "[!]"
	WARNING_PREFIX LogLevel = "[?]"
	INFO_PREFIX    LogLevel = "[i]"
)

func Log(level LogLevel, format string, a ...interface{}) {
	prefixedFormat := string(level) + " " + format
	message := fmt.Sprintf(prefixedFormat, a...)
	log.Println(message)
}
