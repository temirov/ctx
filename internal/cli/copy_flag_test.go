package cli

import (
	"io"
	"testing"

	"github.com/spf13/pflag"
)

func TestRegisterCopyFlagParsesValues(t *testing.T) {
	testCases := []struct {
		name        string
		arguments   []string
		expected    bool
		expectError bool
	}{
		{
			name:        "defaults_to_false",
			arguments:   []string{},
			expected:    false,
			expectError: false,
		},
		{
			name:        "sets_true_without_value",
			arguments:   []string{"--copy"},
			expected:    true,
			expectError: false,
		},
		{
			name:        "sets_false_with_equals",
			arguments:   []string{"--copy=false"},
			expected:    false,
			expectError: false,
		},
		{
			name:        "sets_false_with_no",
			arguments:   []string{"--copy", "no"},
			expected:    false,
			expectError: false,
		},
		{
			name:        "rejects_invalid_text",
			arguments:   []string{"--copy", "maybe"},
			expected:    false,
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var flagValue bool
			flagSet := pflag.NewFlagSet("copy-flag", pflag.ContinueOnError)
			flagSet.SetOutput(io.Discard)
			registerCopyFlag(flagSet, &flagValue)
			normalizedArguments := normalizeCopyFlagArguments(testCase.arguments)
			parseErr := flagSet.Parse(normalizedArguments)
			if testCase.expectError {
				if parseErr == nil {
					t.Fatalf("expected error for arguments %v", testCase.arguments)
				}
				return
			}
			if parseErr != nil {
				t.Fatalf("unexpected parse error: %v", parseErr)
			}
			if flagValue != testCase.expected {
				t.Fatalf("expected value %t, got %t", testCase.expected, flagValue)
			}
		})
	}
}
