package format

import (
	"fmt"
	"log"
)

type LogLevel string

const (
	HEADER_PREFIX  LogLevel = "[+]"
	RUN_PREFIX     LogLevel = "[-]"
	ERROR_PREFIX   LogLevel = "[E]"
	WARNING_PREFIX LogLevel = "[W]"
	INFO_PREFIX    LogLevel = "[I]"
)

func Log(level LogLevel, format string, a ...any) {
	prefixedFormat := string(level) + " " + format
	message := fmt.Sprintf(prefixedFormat, a...)
	log.Println(message)
}
