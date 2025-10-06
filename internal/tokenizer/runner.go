package tokenizer

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type scriptCounter struct {
	runner     string
	scriptPath string
	args       []string
	helperName string
	timeout    time.Duration
}

func (counter scriptCounter) Name() string {
	return counter.helperName
}

func (counter scriptCounter) CountString(input string) (int, error) {
	runner := strings.TrimSpace(counter.runner)
	if runner == "" {
		return 0, fmt.Errorf("uv executable not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), counter.timeout)
	defer cancel()

	commandArgs := append([]string{"run", counter.scriptPath}, counter.args...)
	command := exec.CommandContext(ctx, runner, commandArgs...)
	command.Stdin = strings.NewReader(input)

	outputBytes, err := command.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return 0, fmt.Errorf("uv helper timeout: %w", ctx.Err())
	}
	if err != nil {
		return 0, fmt.Errorf("uv helper error: %v, output: %s", err, strings.TrimSpace(string(outputBytes)))
	}

	tokenText := strings.TrimSpace(string(outputBytes))
	if tokenText == "" {
		return 0, fmt.Errorf("uv helper returned empty output")
	}

	tokenCount, parseErr := strconv.Atoi(tokenText)
	if parseErr != nil {
		return 0, fmt.Errorf("unexpected uv helper output: %q", tokenText)
	}
	return tokenCount, nil
}
