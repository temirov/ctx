package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type javaScriptDetector struct {
	client npmRegistryClient
}

func newJavaScriptDetector(client npmRegistryClient) javaScriptDetector {
	if client == nil {
		client = newNPMRegistry("")
	}
	return javaScriptDetector{client: client}
}

func (detector javaScriptDetector) Ecosystem() Ecosystem {
	return EcosystemJavaScript
}

func (detector javaScriptDetector) Detect(ctx context.Context, rootPath string, options Options) ([]Dependency, error) {
	manifestPath := filepath.Join(rootPath, "package.json")
	data, readErr := os.ReadFile(manifestPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return nil, nil
		}
		return nil, fmt.Errorf("read package.json: %w", readErr)
	}
	var manifest npmManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}
	var dependencies []Dependency
	seen := map[string]struct{}{}
	appendDependency := func(name string, version string) {
		if name == "" || version == "" {
			return
		}
		if _, exists := seen[name]; exists {
			return
		}
		metadata, metadataErr := detector.client.Metadata(ctx, name)
		if metadataErr != nil {
			return
		}
		owner, repository := parseGitHubRepository(metadata.RepositoryURL)
		if owner == "" || repository == "" {
			owner, repository = parseGitHubRepository(metadata.Homepage)
			if owner == "" {
				return
			}
		}
		source := RepositorySource{
			Owner:      owner,
			Repository: repository,
			Reference:  "main",
			DocPaths:   defaultDocPaths(),
		}
		dependency, dependencyErr := NewDependency(name, version, EcosystemJavaScript, source)
		if dependencyErr != nil {
			return
		}
		if dependencies == nil {
			dependencies = []Dependency{}
		}
		seen[name] = struct{}{}
		dependencies = append(dependencies, dependency)
	}
	for name, version := range manifest.Dependencies {
		appendDependency(name, version)
	}
	if options.IncludeDev {
		for name, version := range manifest.DevDependencies {
			appendDependency(name, version)
		}
	}
	return dependencies, nil
}

type npmManifest struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func parseGitHubRepository(raw string) (string, string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ""
	}
	trimmed = strings.TrimPrefix(trimmed, "git+")
	trimmed = strings.TrimSuffix(trimmed, ".git")
	if !strings.Contains(trimmed, "github.com") {
		return "", ""
	}
	trimmed = strings.TrimPrefix(trimmed, "github:")
	trimmed = strings.TrimPrefix(trimmed, "git://")
	trimmed = strings.TrimPrefix(trimmed, "https://")
	if !strings.HasPrefix(trimmed, "github.com") {
		trimmed = "github.com/" + trimmed
	}
	segments := strings.Split(strings.Trim(trimmed, "/"), "/")
	index := 0
	for index < len(segments) && segments[index] != "github.com" {
		index++
	}
	if index >= len(segments)-2 {
		return "", ""
	}
	owner := segments[index+1]
	repository := segments[index+2]
	if owner == "" || repository == "" {
		return "", ""
	}
	return owner, repository
}
