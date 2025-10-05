package tokenizer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type pythonCounter struct {
	executable string
	scriptPath string
	args       []string
	helperName string
	timeout    time.Duration
}

func (counter pythonCounter) Name() string {
	return counter.helperName
}

func (counter pythonCounter) CountString(input string) (int, error) {
	execPath := counter.executable
	if execPath == "" {
		execPath = pythonExecutableGuess
	}
	ctx, cancel := context.WithTimeout(context.Background(), counter.timeout)
	defer cancel()

	commandArgs := append([]string{counter.scriptPath}, counter.args...)
	command := exec.CommandContext(ctx, execPath, commandArgs...)

	stdinPipe, stdinErr := command.StdinPipe()
	if stdinErr != nil {
		return 0, stdinErr
	}
	stdoutPipe, stdoutErr := command.StdoutPipe()
	if stdoutErr != nil {
		return 0, stdoutErr
	}
	stderrPipe, stderrErr := command.StderrPipe()
	if stderrErr != nil {
		return 0, stderrErr
	}

	if startErr := command.Start(); startErr != nil {
		return 0, startErr
	}

	writer := bufio.NewWriter(stdinPipe)
	if _, writeErr := writer.WriteString(input); writeErr != nil {
		_ = stdinPipe.Close()
		_ = command.Wait()
		return 0, writeErr
	}
	if flushErr := writer.Flush(); flushErr != nil {
		_ = stdinPipe.Close()
		_ = command.Wait()
		return 0, flushErr
	}
	_ = stdinPipe.Close()

	outputBytes, readErr := io.ReadAll(stdoutPipe)
	if readErr != nil {
		_ = command.Wait()
		return 0, readErr
	}
	errorBytes, _ := io.ReadAll(stderrPipe)

	waitErr := command.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		return 0, fmt.Errorf("python helper timeout: %w", ctx.Err())
	}
	if waitErr != nil {
		return 0, fmt.Errorf("python helper error: %v, stderr: %s", waitErr, strings.TrimSpace(string(errorBytes)))
	}

	tokenText := strings.TrimSpace(string(outputBytes))
	if tokenText == "" {
		if len(errorBytes) > 0 {
			return 0, fmt.Errorf("python helper empty output, stderr: %s", strings.TrimSpace(string(errorBytes)))
		}
		return 0, fmt.Errorf("python helper empty output")
	}

	tokenCount, parseErr := strconv.Atoi(tokenText)
	if parseErr != nil {
		return 0, fmt.Errorf("unexpected python output: %q, stderr: %q", tokenText, strings.TrimSpace(string(errorBytes)))
	}
	return tokenCount, nil
}
