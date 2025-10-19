package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRegisterBooleanFlagParsesValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		defaultValue bool
		arguments    []string
		expected     bool
		expectError  bool
	}{
		{
			name:         "defaults_to_false",
			defaultValue: false,
			arguments:    []string{},
			expected:     false,
			expectError:  false,
		},
		{
			name:         "sets_true_without_value",
			defaultValue: false,
			arguments:    []string{"--feature"},
			expected:     true,
			expectError:  false,
		},
		{
			name:         "sets_false_with_equals",
			defaultValue: true,
			arguments:    []string{"--feature=false"},
			expected:     false,
			expectError:  false,
		},
		{
			name:         "sets_false_with_no_literal",
			defaultValue: true,
			arguments:    []string{"--feature", "no"},
			expected:     false,
			expectError:  false,
		},
		{
			name:         "sets_true_with_on_literal",
			defaultValue: false,
			arguments:    []string{"--feature", "on"},
			expected:     true,
			expectError:  false,
		},
		{
			name:         "ignores_non_boolean_trailing_value",
			defaultValue: false,
			arguments:    []string{"--feature", "maybe"},
			expected:     true,
			expectError:  false,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			command := &cobra.Command{Use: "boolean-test"}
			flagSet := command.Flags()
			flagValue := !testCase.defaultValue
			registerBooleanFlag(flagSet, &flagValue, "feature", testCase.defaultValue, "toggle feature behaviour")
			normalizedArguments := normalizeBooleanFlagArguments(command, testCase.arguments)
			parseErr := command.ParseFlags(normalizedArguments)
			if testCase.expectError {
				if parseErr == nil {
					t.Fatalf("expected parse error for arguments %v", testCase.arguments)
				}
				return
			}
			if parseErr != nil {
				t.Fatalf("unexpected parse error: %v", parseErr)
			}
			if len(testCase.arguments) == 0 && flagValue != testCase.defaultValue {
				t.Fatalf("expected default %t, got %t", testCase.defaultValue, flagValue)
			}
			if flagValue != testCase.expected {
				t.Fatalf("expected %t, got %t", testCase.expected, flagValue)
			}
		})
	}
}
