package discover

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type stubNPMRegistry struct {
	metadata map[string]npmPackageMetadata
	err      error
}

func (registry stubNPMRegistry) Metadata(ctx context.Context, name string) (npmPackageMetadata, error) {
	if registry.err != nil {
		return npmPackageMetadata{}, registry.err
	}
	if metadata, ok := registry.metadata[name]; ok {
		return metadata, nil
	}
	return npmPackageMetadata{}, nil
}

func TestJavaScriptDetectorIncludesDevDependenciesWhenEnabled(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "package.json")
	if err := os.WriteFile(manifestPath, []byte(`{
  "name": "example",
  "devDependencies": {
    "playwright": "^1.0.0"
  }
}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	registry := stubNPMRegistry{
		metadata: map[string]npmPackageMetadata{
			"playwright": {
				RepositoryURL: "https://github.com/microsoft/playwright",
			},
		},
	}
	detector := newJavaScriptDetector(registry)
	dependencies, err := detector.Detect(context.Background(), tempDir, Options{IncludeDev: true})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(dependencies))
	}
	if dependencies[0].Name != "playwright" {
		t.Fatalf("unexpected dependency %+v", dependencies[0])
	}
}

func TestJavaScriptDetectorFallsBackToDevDependenciesWhenRuntimeMissing(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(`{
  "name": "example",
  "devDependencies": {
    "vite": "^5.0.0"
  }
}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	detector := newJavaScriptDetector(stubNPMRegistry{
		metadata: map[string]npmPackageMetadata{
			"vite": {
				RepositoryURL: "https://github.com/vitejs/vite",
			},
		},
	})
	result, err := detector.Detect(context.Background(), tempDir, Options{})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(result) != 1 || result[0].Name != "vite" {
		t.Fatalf("expected fallback dependency, got %+v", result)
	}
}

func TestJavaScriptDetectorSkipsDevDependenciesWhenRuntimePresent(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(`{
  "name": "example",
  "dependencies": {
    "alpinejs": "^3.13.5"
  },
  "devDependencies": {
    "eslint": "^8.57.0"
  }
}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	detector := newJavaScriptDetector(stubNPMRegistry{
		metadata: map[string]npmPackageMetadata{
			"alpinejs": {
				RepositoryURL: "https://github.com/alpinejs/alpine",
			},
			"eslint": {
				RepositoryURL: "https://github.com/eslint/eslint",
			},
		},
	})
	result, err := detector.Detect(context.Background(), tempDir, Options{})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(result) != 1 || result[0].Name != "alpinejs" {
		t.Fatalf("expected only runtime dependency, got %+v", result)
	}
}
