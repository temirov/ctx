package utils_test

import (
	"reflect"
	"testing"

	"github.com/temirov/ctx/utils"
)

const (
	patternAlpha = "alpha"
	patternBeta  = "beta"
	patternGamma = "gamma"
)

// TestDeduplicatePatterns verifies removal of duplicate patterns while preserving order.
func TestDeduplicatePatterns(testingHandle *testing.T) {
	testCases := []struct {
		name             string
		patterns         []string
		expectedPatterns []string
	}{
		{
			name:             "NilInput",
			patterns:         nil,
			expectedPatterns: []string{},
		},
		{
			name:             "EmptyInput",
			patterns:         []string{},
			expectedPatterns: []string{},
		},
		{
			name:             "NoDuplicates",
			patterns:         []string{patternAlpha, patternBeta, patternGamma},
			expectedPatterns: []string{patternAlpha, patternBeta, patternGamma},
		},
		{
			name:             "WithDuplicates",
			patterns:         []string{patternAlpha, patternBeta, patternAlpha, patternGamma, patternBeta},
			expectedPatterns: []string{patternAlpha, patternBeta, patternGamma},
		},
		{
			name:             "AllDuplicates",
			patterns:         []string{patternAlpha, patternAlpha},
			expectedPatterns: []string{patternAlpha},
		},
	}

	for _, testCase := range testCases {
		testingHandle.Run(testCase.name, func(testingHandle *testing.T) {
			result := utils.DeduplicatePatterns(testCase.patterns)
			if !reflect.DeepEqual(result, testCase.expectedPatterns) {
				testingHandle.Fatalf("expected %v, got %v", testCase.expectedPatterns, result)
			}
		})
	}
}
