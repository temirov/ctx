package utils_test

import (
	"testing"
	"time"

	"github.com/tyemirov/ctx/internal/utils"
)

func TestFormatFileSize(t *testing.T) {
	testCases := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{name: "negative", bytes: -1, expected: "0b"},
		{name: "zero", bytes: 0, expected: "0b"},
		{name: "bytes", bytes: 512, expected: "512b"},
		{name: "one kilobyte", bytes: 1024, expected: "1kb"},
		{name: "fractional kilobyte", bytes: 1536, expected: "1.5kb"},
		{name: "ten megabytes", bytes: 10 * 1024 * 1024, expected: "10mb"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := utils.FormatFileSize(testCase.bytes)
			if result != testCase.expected {
				t.Fatalf("expected %s, got %s", testCase.expected, result)
			}
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	location := time.Now().Location()
	testCases := []struct {
		name     string
		value    time.Time
		expected string
	}{
		{
			name:     "zero time",
			value:    time.Time{},
			expected: "",
		},
		{
			name:     "local timestamp",
			value:    time.Date(2024, time.January, 2, 15, 4, 0, 0, location),
			expected: "2024-01-02 15:04",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := utils.FormatTimestamp(testCase.value)
			if result != testCase.expected {
				t.Fatalf("expected %s, got %s", testCase.expected, result)
			}
		})
	}
}
