package tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

const docCommandName = "doc"

func TestDocCommandGitHubExtraction(t *testing.T) {
	if testing.Short() {
		t.Skip("skip doc extraction tests in short mode")
	}

	binary := buildBinary(t)
	testCases := []struct {
		name       string
		owner      string
		repository string
		reference  string
		rootPath   string
		mode       string
		files      map[string]string
		expect     []string
		unexpected []string
	}{
		{
			name:       "jspreadsheet_full",
			owner:      "jspreadsheet",
			repository: "ce",
			reference:  "main",
			rootPath:   "docs/jspreadsheet",
			mode:       "full",
			files: map[string]string{
				"docs/jspreadsheet/editors.md": "# Editors\n\nEditors overview.",
				"docs/jspreadsheet/filters.md": "# Filters\n\nFilters overview.",
			},
			expect: []string{
				"# Documentation for jspreadsheet/ce (main)",
				"## editors.md",
				"Editors overview.",
				"## filters.md",
				"Filters overview.",
			},
		},
		{
			name:       "jspreadsheet_relevant",
			owner:      "jspreadsheet",
			repository: "ce",
			reference:  "main",
			rootPath:   "docs/jspreadsheet",
			mode:       "relevant",
			files: map[string]string{
				"docs/jspreadsheet/editors.md": "# Editors\n\nEditors overview.",
				"docs/jspreadsheet/filters.md": "# Filters\n\nFilters overview.",
			},
			expect: []string{
				"# Documentation for jspreadsheet/ce (main)",
				"## editors.md",
				"Editors overview.",
			},
			unexpected: []string{
				"Filters overview.",
			},
		},
		{
			name:       "marked_full",
			owner:      "markedjs",
			repository: "marked",
			reference:  "main",
			rootPath:   "docs",
			mode:       "full",
			files: map[string]string{
				"docs/api.md":   "# API\n\nAPI usage.",
				"docs/guide.md": "# Guide\n\nGetting started guide.",
			},
			expect: []string{
				"# Documentation for markedjs/marked (main)",
				"API usage.",
				"Getting started guide.",
			},
		},
		{
			name:       "beercss_full",
			owner:      "beercss",
			repository: "beercss",
			reference:  "main",
			rootPath:   "docs",
			mode:       "full",
			files: map[string]string{
				"docs/components.md": "# Components\n\nComponent overview.",
				"docs/layouts.md":    "# Layouts\n\nLayout overview.",
			},
			expect: []string{
				"# Documentation for beercss/beercss (main)",
				"Component overview.",
				"Layout overview.",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			server := startGitHubMockServer(t, testCase.owner, testCase.repository, testCase.reference, testCase.rootPath, testCase.files)
			defer server.Close()

			workingDirectory := t.TempDir()
			arguments := []string{
				docCommandName,
				"--owner", testCase.owner,
				"--repo", testCase.repository,
				"--path", testCase.rootPath,
				"--api-base", server.URL,
				fmt.Sprintf("--doc=%s", testCase.mode),
			}
			if testCase.reference != "" {
				arguments = append(arguments, "--ref", testCase.reference)
			}

			output := runCommand(t, binary, arguments, workingDirectory)
			for _, expected := range testCase.expect {
				if !strings.Contains(output, expected) {
					t.Fatalf("expected output to contain %q\n%s", expected, output)
				}
			}
			for _, unexpected := range testCase.unexpected {
				if strings.Contains(output, unexpected) {
					t.Fatalf("expected output to exclude %q\n%s", unexpected, output)
				}
			}
			if strings.Contains(output, "## Go to") || strings.Contains(output, "[Begin](INDEX.md)") {
				t.Fatalf("navigation block should be removed\n%s", output)
			}
		})
	}
}

func startGitHubMockServer(t *testing.T, owner string, repository string, reference string, rootPath string, files map[string]string) *httptest.Server {
	t.Helper()
	normalizedRoot := strings.Trim(strings.TrimSpace(rootPath), "/")
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if reference != "" && request.URL.Query().Get("ref") != reference {
			http.NotFound(writer, request)
			return
		}
		prefix := fmt.Sprintf("/repos/%s/%s/contents/", owner, repository)
		if !strings.HasPrefix(request.URL.Path, prefix) {
			http.NotFound(writer, request)
			return
		}
		requested := strings.Trim(strings.TrimPrefix(request.URL.Path, prefix), "/")
		if strings.Contains(requested, "%") {
			t.Fatalf("unexpected escaped segment in %s", requested)
		}
		if requested == "" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		if requested == normalizedRoot {
			var entries []map[string]interface{}
			for filePath := range files {
				if !strings.HasPrefix(filePath, normalizedRoot+"/") {
					continue
				}
				relative := strings.TrimPrefix(filePath, normalizedRoot+"/")
				if strings.Contains(relative, "/") {
					continue
				}
				entries = append(entries, map[string]interface{}{
					"name": filepath.Base(filePath),
					"path": filePath,
					"type": "file",
				})
			}
			if err := json.NewEncoder(writer).Encode(entries); err != nil {
				t.Fatalf("encode directory listing: %v", err)
			}
			return
		}
		content, ok := files[requested]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		payload := map[string]interface{}{
			"name":     filepath.Base(requested),
			"path":     requested,
			"type":     "file",
			"encoding": "base64",
			"content":  base64.StdEncoding.EncodeToString([]byte(content)),
		}
		if err := json.NewEncoder(writer).Encode(payload); err != nil {
			t.Fatalf("encode file payload: %v", err)
		}
	})
	return httptest.NewServer(handler)
}
