package sensors

//nolint // maintainability: highly cohesive logic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"
)

func handleStartError(name string, err error) error {
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("%s not found in PATH", name)
	}
	return fmt.Errorf("failed to start %s: %w", name, err)
}

func getExitCodeAndError(name string, err error, stderrBuf *bytes.Buffer) (int, error) {
	if err == nil {
		return 0, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), nil
	}
	return 0, fmt.Errorf("failed to run %s: %w", name, err)
}

func runLintCommandJSON(name string, target interface{}, args ...string) (int, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return 0, nil, handleStartError(name, err)
	}

	decodeErr := json.NewDecoder(stdout).Decode(target)

	err = cmd.Wait()

	exitCode, runErr := getExitCodeAndError(name, err, &stderrBuf)
	if runErr != nil {
		return 0, stderrBuf.Bytes(), runErr
	}

	if decodeErr != nil && decodeErr != io.EOF {
		return exitCode, stderrBuf.Bytes(), fmt.Errorf("failed to decode JSON: %w", decodeErr)
	}

	return exitCode, stderrBuf.Bytes(), nil
}

func checkLintExecutionError(name string, exitCode int, output []byte, err error) error {
	if err != nil {
		if exitCode > 0 {
			return fmt.Errorf("%s crashed or encountered a configuration error (exit code %d): %v\n%s", name, exitCode, err, string(output))
		}
		return fmt.Errorf("%s error: %w", name, err)
	}
	return nil
}
