package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runCmd executes a command and streams stdout/stderr to the current process.
func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

// runCmdOutput executes a command and returns captured stdout.
func runCmdOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("command failed: %s %s: %w", name, strings.Join(args, " "), err)
	}
	return out.String(), nil
}

// runCmdAllowFail executes a command and returns its raw error without wrapping.
func runCmdAllowFail(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
