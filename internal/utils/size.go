package utils

import (
	"fmt"
	"strings"
)

// FormatFileSize converts a byte length into a human-readable lower-case unit string.
func FormatFileSize(bytes int64) string {
	if bytes < 0 {
		return "0b"
	}
	units := []string{"b", "kb", "mb", "gb", "tb", "pb"}
	value := float64(bytes)
	unitIndex := 0
	for value >= 1024 && unitIndex < len(units)-1 {
		value /= 1024
		unitIndex++
	}
	if unitIndex == 0 {
		return fmt.Sprintf("%db", bytes)
	}
	if value < 10 {
		formatted := fmt.Sprintf("%.1f", value)
		formatted = strings.TrimSuffix(formatted, ".0")
		return formatted + units[unitIndex]
	}
	return fmt.Sprintf("%.0f%s", value, units[unitIndex])
}
