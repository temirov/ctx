package cli

import (
	"reflect"
	"testing"
)

func TestResolveGitHubAuthorizationToken(t *testing.T) {
	testCases := []struct {
		name          string
		primaryValue  string
		fallbackValue string
		expectedToken string
	}{
		{
			name:          "primary token wins",
			primaryValue:  " primary-token ",
			fallbackValue: "fallback-token",
			expectedToken: "primary-token",
		},
		{
			name:          "fallback token used when primary empty",
			primaryValue:  "   ",
			fallbackValue: " fallback-token ",
			expectedToken: "fallback-token",
		},
		{
			name:          "no tokens available",
			primaryValue:  "",
			fallbackValue: "",
			expectedToken: "",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv(githubTokenEnvPrimary, testCase.primaryValue)
			t.Setenv(githubTokenEnvFallback, testCase.fallbackValue)
			result := resolveGitHubAuthorizationToken()
			if result != testCase.expectedToken {
				t.Fatalf("expected token %q, got %q", testCase.expectedToken, result)
			}
		})
	}
}

func TestParseGitHubRepositoryURL(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    repositoryCoordinates
		expectError bool
	}{
		{
			name:  "tree path with explicit branch",
			input: "https://github.com/example/project/tree/main/docs",
			expected: repositoryCoordinates{
				Owner:      "example",
				Repository: "project",
				Reference:  "main",
				RootPath:   "docs",
			},
		},
		{
			name:  "blob path with branch and nested directory",
			input: "https://github.com/jspreadsheet/ce/blob/master/docs/jspreadsheet/docs",
			expected: repositoryCoordinates{
				Owner:      "jspreadsheet",
				Repository: "ce",
				Reference:  "master",
				RootPath:   "docs/jspreadsheet/docs",
			},
		},
		{
			name:     "empty keeps defaults",
			input:    "",
			expected: repositoryCoordinates{},
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			result, err := parseGitHubRepositoryURL(testCase.input)
			if testCase.expectError {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if !reflect.DeepEqual(result, testCase.expected) {
				t.Fatalf("expected %+v, got %+v", testCase.expected, result)
			}
		})
	}
}
