package config

import (
	"bufio"
	"os"
	"strings"
)

// LoadContentIgnore reads the .contentignore file (if it exists) and returns a slice of ignore patterns.
// Blank lines and lines beginning with '#' are skipped.
func LoadContentIgnore(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return patterns, nil
}
