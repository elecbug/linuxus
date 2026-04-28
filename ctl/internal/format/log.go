package format

import (
	"fmt"
	"time"
)

// LogLevel defines the severity level of log messages.
type LogLevel string

const (
	RUN_PREFIX     LogLevel = "[RUN]"
	DETAIL_PREFIX  LogLevel = "[DETAIL]"
	ERROR_PREFIX   LogLevel = "[ERROR]"
	WARNING_PREFIX LogLevel = "[WARNING]"
	INFO_PREFIX    LogLevel = "[INFO]"
)

// Log formats a log message with the specified log level and arguments.
func Log(level LogLevel, format string, a ...any) {
	prefixedFormat := getTimestamp() + " " + string(level) + " " + format
	message := fmt.Sprintf(prefixedFormat, a...)
	fmt.Println(message)
}

// getTimestamp returns the current timestamp formatted as a string.
func getTimestamp() string {
	return fmt.Sprintf("%s", fmt.Sprint(time.Now().Format("2006-01-02 15:04:05")))
}
