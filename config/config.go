// Package config loads and parses the .ignore file into a slice of patterns.
package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadContentIgnore reads the .ignore file (if it exists) and returns a slice of ignore patterns.
// Blank lines and lines beginning with '#' are skipped.
//
// #nosec G304: filePath is intentionally user-provided, as we must open .ignore in that directory.
func LoadContentIgnore(filePath string) ([]string, error) {
	fileHandle, openError := os.Open(filePath)
	if openError != nil {
		return nil, openError
	}
	defer func() {
		closeErr := fileHandle.Close()
		if closeErr != nil {
			// We won't fail the entire operation for a close error, but we log it.
			fmt.Fprintf(os.Stderr, "Warning: failed to close %s: %v\n", filePath, closeErr)
		}
	}()

	var patterns []string
	scanner := bufio.NewScanner(fileHandle)
	for scanner.Scan() {
		lineValue := strings.TrimSpace(scanner.Text())
		if lineValue == "" || strings.HasPrefix(lineValue, "#") {
			continue
		}
		patterns = append(patterns, lineValue)
	}
	if scanError := scanner.Err(); scanError != nil {
		return nil, scanError
	}
	return patterns, nil
}
