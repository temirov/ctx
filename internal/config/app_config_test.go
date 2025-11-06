package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tyemirov/ctx/internal/utils"
)

type configTestCase struct {
	name           string
	globalContent  string
	localContent   string
	explicitPath   string
	expectFormat   string
	expectSummary  *bool
	expectTokens   *bool
	expectModel    string
	expectContent  *bool
	expectCopy     *bool
	expectCopyOnly *bool
}

func boolPointer(value bool) *bool {
	pointer := value
	return &pointer
}

func TestLoadApplicationConfigurationMergesSources(t *testing.T) {
	summaryFalse := boolPointer(false)
	tokensEnabled := boolPointer(true)

	testCases := []configTestCase{
		{
			name:          "local_overrides_global",
			globalContent: "tree:\n  format: raw\n  summary: false\n  content: false\n  clipboard: true\n",
			localContent:  "tree:\n  format: xml\n  content: true\n  copy: false\n  tokens:\n    enabled: true\n    model: custom\n",
			expectFormat:  "xml",
			expectSummary: summaryFalse,
			expectTokens:  tokensEnabled,
			expectModel:   "custom",
			expectContent: boolPointer(true),
			expectCopy:    boolPointer(false),
		},
		{
			name:           "explicit_path_only",
			globalContent:  "tree:\n  format: json\n",
			localContent:   "",
			explicitPath:   "custom.yaml",
			expectFormat:   "raw",
			expectSummary:  nil,
			expectTokens:   nil,
			expectModel:    "",
			expectContent:  nil,
			expectCopy:     nil,
			expectCopyOnly: nil,
		},
		{
			name:           "copy_key_applies",
			globalContent:  "tree:\n  copy: true\n",
			localContent:   "",
			expectFormat:   "",
			expectSummary:  nil,
			expectTokens:   nil,
			expectModel:    "",
			expectContent:  nil,
			expectCopy:     boolPointer(true),
			expectCopyOnly: nil,
		},
		{
			name:           "copy_only_key_applies",
			globalContent:  "tree:\n  copy_only: true\n",
			expectFormat:   "",
			expectSummary:  nil,
			expectTokens:   nil,
			expectModel:    "",
			expectContent:  nil,
			expectCopy:     boolPointer(true),
			expectCopyOnly: boolPointer(true),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			homeDir := t.TempDir()
			workingDir := t.TempDir()
			configDir := filepath.Join(homeDir, utils.GlobalConfigDirectoryName)
			if err := os.MkdirAll(configDir, 0o755); err != nil {
				t.Fatalf("create config dir: %v", err)
			}
			if testCase.globalContent != "" {
				globalPath := filepath.Join(configDir, utils.ConfigFileName)
				if err := os.WriteFile(globalPath, []byte(testCase.globalContent), 0o600); err != nil {
					t.Fatalf("write global config: %v", err)
				}
			}
			localPath := filepath.Join(workingDir, utils.ConfigFileName)
			if testCase.localContent != "" {
				if err := os.WriteFile(localPath, []byte(testCase.localContent), 0o600); err != nil {
					t.Fatalf("write local config: %v", err)
				}
			}
			explicitPath := testCase.explicitPath
			if explicitPath != "" {
				target := filepath.Join(workingDir, explicitPath)
				if err := os.WriteFile(target, []byte("tree:\n  format: raw\n"), 0o600); err != nil {
					t.Fatalf("write explicit config: %v", err)
				}
			}

			t.Setenv("HOME", homeDir)
			t.Setenv("USERPROFILE", homeDir)

			loadedConfig, err := LoadApplicationConfiguration(LoadOptions{
				WorkingDirectory: workingDir,
				ExplicitFilePath: testCase.explicitPath,
			})
			if err != nil {
				t.Fatalf("LoadApplicationConfiguration error: %v", err)
			}

			if loadedConfig.Tree.Format != testCase.expectFormat {
				t.Fatalf("expected format %s, got %s", testCase.expectFormat, loadedConfig.Tree.Format)
			}
			if testCase.expectSummary == nil {
				if loadedConfig.Tree.Summary != nil {
					t.Fatalf("expected no summary override")
				}
			} else {
				if loadedConfig.Tree.Summary == nil || *loadedConfig.Tree.Summary != *testCase.expectSummary {
					t.Fatalf("unexpected summary value")
				}
			}
			if testCase.expectTokens == nil {
				if loadedConfig.Tree.Tokens.Enabled != nil {
					t.Fatalf("expected no tokens override")
				}
			} else {
				if loadedConfig.Tree.Tokens.Enabled == nil || *loadedConfig.Tree.Tokens.Enabled != *testCase.expectTokens {
					t.Fatalf("unexpected tokens enabled value")
				}
			}
			if loadedConfig.Tree.Tokens.Model != testCase.expectModel {
				t.Fatalf("expected model %q, got %q", testCase.expectModel, loadedConfig.Tree.Tokens.Model)
			}
			if testCase.expectContent == nil {
				if loadedConfig.Tree.IncludeContent != nil {
					t.Fatalf("expected no content override")
				}
			} else {
				if loadedConfig.Tree.IncludeContent == nil || *loadedConfig.Tree.IncludeContent != *testCase.expectContent {
					t.Fatalf("unexpected content value")
				}
			}
			if testCase.expectCopy == nil {
				if loadedConfig.Tree.Copy != nil {
					t.Fatalf("expected no copy override")
				}
			} else {
				if loadedConfig.Tree.Copy == nil || *loadedConfig.Tree.Copy != *testCase.expectCopy {
					t.Fatalf("unexpected copy value")
				}
			}
			if testCase.expectCopyOnly == nil {
				if loadedConfig.Tree.CopyOnly != nil {
					t.Fatalf("expected no copy_only override")
				}
			} else {
				if loadedConfig.Tree.CopyOnly == nil || *loadedConfig.Tree.CopyOnly != *testCase.expectCopyOnly {
					t.Fatalf("unexpected copy_only value")
				}
			}
		})
	}
}

func TestStreamCopySettingsEnforcesCopyOnlyImpliesCopy(t *testing.T) {
	configuration := StreamCommandConfiguration{
		CopyOnly: boolPointer(true),
	}
	settings := configuration.CopySettings()
	if settings.Copy == nil || !*settings.Copy {
		t.Fatalf("expected copy to be enabled when copyOnly is true")
	}
	if settings.CopyOnly == nil || !*settings.CopyOnly {
		t.Fatalf("expected copyOnly to remain true")
	}
}

func TestCallChainCopySettingsIncludesLegacyClipboard(t *testing.T) {
	configuration := CallChainConfiguration{
		LegacyClipboard: boolPointer(true),
	}
	settings := configuration.CopySettings()
	if settings.Copy == nil || !*settings.Copy {
		t.Fatalf("expected legacy clipboard to enable copy")
	}
}

func TestStreamCommandMergePreservesCopyOnlyInvariant(t *testing.T) {
	base := StreamCommandConfiguration{}
	override := StreamCommandConfiguration{CopyOnly: boolPointer(true)}
	merged := base.merge(override)
	settings := merged.CopySettings()
	if settings.Copy == nil || !*settings.Copy {
		t.Fatalf("expected copy to be enforced when copyOnly is true")
	}
}
