package cli

import (
	"reflect"
	"testing"
)

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

func TestResolveRepositoryCoordinatesAcceptsUnifiedPath(t *testing.T) {
	result, err := resolveRepositoryCoordinates("jspreadsheet/ce/docs/jspreadsheet", "", "", "", "")
	if err != nil {
		t.Fatalf("expected unified path to resolve without error, got %v", err)
	}
	expected := repositoryCoordinates{
		Owner:      "jspreadsheet",
		Repository: "ce",
		Reference:  "",
		RootPath:   "docs/jspreadsheet",
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %+v, got %+v", expected, result)
	}
}

func TestResolveRepositoryCoordinatesDefaultsRootForOwnerRepoFormat(t *testing.T) {
	result, err := resolveRepositoryCoordinates("example/documentation", "", "", "", "")
	if err != nil {
		t.Fatalf("expected owner/repo format to resolve without error, got %v", err)
	}
	expected := repositoryCoordinates{
		Owner:      "example",
		Repository: "documentation",
		Reference:  "",
		RootPath:   ".",
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %+v, got %+v", expected, result)
	}
}
