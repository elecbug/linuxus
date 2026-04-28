package format

import (
	"bytes"
	"fmt"
	"io"
	"strings"
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

// DockerBuildLog processes and logs the output from a Docker build operation.
func DockerBuildLog(level LogLevel, logBuf bytes.Buffer, imageName string) error {
	Log(level, "Building image %s...", imageName)

	for logBuf.Len() > 0 {
		line, err := logBuf.ReadString('\n')
		line = strings.TrimSpace(strings.Trim(line, "\r\n"))

		if err != nil && err != io.EOF {
			Log(ERROR_PREFIX, "Error reading Docker build log: %v", err)
			return err
		}

		if len(line) == 0 {
			continue
		} else if !strings.HasPrefix(line, "--->") {
			Log(level, line)
		}
	}

	return nil
}

// getTimestamp returns the current timestamp formatted as a string.
func getTimestamp() string {
	return fmt.Sprintf("%s", fmt.Sprint(time.Now().Format("2006-01-02 15:04:05")))
}
