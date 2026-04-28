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

// DockerBuildLog reads and formats Docker build logs with the specified log level and format.
func DockerBuildLog(level LogLevel, logBuf bytes.Buffer, imageName string) error {
	Log(level, "Building image %s...", imageName)

	for logBuf.Len() > 0 {
		line, err := logBuf.ReadString('\n')
		line = strings.TrimSpace(line)

		if err != nil && err != io.EOF {
			fmt.Printf("\n")
			Log(ERROR_PREFIX, "Error reading build logs: %v", err)

			return err
		}
		if strings.HasPrefix(line, "Step ") ||
			strings.HasPrefix(line, "Successfully") {

			Log(level, line)
		} else {
			//
		}
	}

	return nil
}

// getTimestamp returns the current timestamp formatted as a string.
func getTimestamp() string {
	return fmt.Sprintf("%s", fmt.Sprint(time.Now().Format("2006-01-02 15:04:05")))
}
