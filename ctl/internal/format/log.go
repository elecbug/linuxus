package format

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// LogLevel defines the severity level of log messages.
type LogLevel string

const (
	RUN_PREFIX     LogLevel = "[RUN]"
	DETAIL_PREFIX  LogLevel = "[DETAIL]"
	ERROR_PREFIX   LogLevel = "[ERROR]"
	WARNING_PREFIX LogLevel = "[WARNING]"
	INFO_PREFIX    LogLevel = "[INFO]"
	INPUT_PREFIX   LogLevel = "[INPUT]"
)

// Log formats a log message with the specified log level and arguments.
func Log(level LogLevel, format string, a ...any) {
	prefixedFormat := getTimestamp() + " " + string(level) + " " + format
	message := fmt.Sprintf(prefixedFormat, a...)
	fmt.Println(message)
}

// Input prompts the user for a line of text and returns the trimmed result.
// It returns an error if reading fails or if EOF is reached before any input.
func Input(prompt string, a ...any) (string, error) {
	formattedPrompt := fmt.Sprintf(string(INPUT_PREFIX)+" "+prompt, a...)
	fmt.Print(formattedPrompt)

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF && line != "" {
			// valid input at EOF (no trailing newline)
		} else {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// InputPassword prompts the user for a password without echoing the input to the terminal.
// It returns an error if reading fails.
func InputPassword(prompt string, a ...any) (string, error) {
	formattedPrompt := fmt.Sprintf(string(INPUT_PREFIX)+" "+prompt, a...)
	fmt.Print(formattedPrompt)

	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	return string(password), nil
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
