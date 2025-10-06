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
	cleanOutput := sanitizeHelperOutput(string(outputBytes))
	if ctx.Err() == context.DeadlineExceeded {
		return 0, fmt.Errorf("uv helper timeout: %w", ctx.Err())
	}
	if err != nil {
		if cleanOutput != "" {
			return 0, fmt.Errorf("uv helper error: %v, output: %s", err, cleanOutput)
		}
		return 0, fmt.Errorf("uv helper error: %v", err)
	}

	tokenCount, parseErr := parseHelperTokenOutput(cleanOutput)
	if parseErr != nil {
		return 0, parseErr
	}
	return tokenCount, nil
}

func parseHelperTokenOutput(rawOutput string) (int, error) {
	trimmed := strings.TrimSpace(rawOutput)
	if trimmed == "" {
		return 0, fmt.Errorf("uv helper returned empty output")
	}

	lines := strings.Split(trimmed, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(lines[i])
		if candidate == "" {
			continue
		}
		if tokenCount, err := strconv.Atoi(candidate); err == nil {
			return tokenCount, nil
		}
	}

	return 0, fmt.Errorf("unexpected uv helper output: %q", trimmed)
}

func sanitizeHelperOutput(rawOutput string) string {
	if rawOutput == "" {
		return ""
	}

	lines := strings.Split(rawOutput, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isHelperNoiseLine(trimmed) {
			continue
		}
		filtered = append(filtered, trimmed)
	}

	return strings.Join(filtered, "\n")
}

func isHelperNoiseLine(line string) bool {
	if strings.HasPrefix(line, "Installed ") && strings.Contains(line, " packages in ") {
		return true
	}
	return false
}
