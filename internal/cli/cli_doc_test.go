package cli

import (
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
