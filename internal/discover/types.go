package discover

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	errInvalidDependency      = errors.New("invalid dependency")
	errUnsupportedEcosystem   = errors.New("unsupported ecosystem")
	errMissingRepositoryOwner = errors.New("repository owner is required")
	errMissingRepositoryName  = errors.New("repository name is required")
)

// Ecosystem identifies the package manager or language family.
type Ecosystem string

const (
	// EcosystemGo represents Go modules.
	EcosystemGo Ecosystem = "go"
	// EcosystemJavaScript represents npm-based projects.
	EcosystemJavaScript Ecosystem = "js"
	// EcosystemPython represents Python packages.
	EcosystemPython Ecosystem = "python"
)

// Status describes the outcome for a dependency.
type Status string

const (
	// StatusWritten indicates documentation was fetched and written.
	StatusWritten Status = "written"
	// StatusSkipped indicates the dependency could not be processed (e.g., missing docs).
	StatusSkipped Status = "skipped"
	// StatusFailed indicates a hard failure while processing the dependency.
	StatusFailed Status = "failed"
)

// RepositorySource captures the repository coordinates for documentation retrieval.
type RepositorySource struct {
	Owner      string
	Repository string
	Reference  string
	DocPaths   []string
}

// Dependency represents a detected third-party dependency.
type Dependency struct {
	Name      string
	Version   string
	Ecosystem Ecosystem
	Source    RepositorySource
	Metadata  map[string]string
}

// NewDependency constructs a dependency with validation.
func NewDependency(name string, version string, ecosystem Ecosystem, source RepositorySource) (Dependency, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Dependency{}, fmt.Errorf("%w: name is required", errInvalidDependency)
	}
	if ecosystem == "" {
		return Dependency{}, fmt.Errorf("%w: ecosystem is required", errInvalidDependency)
	}
	if source.Owner == "" {
		return Dependency{}, fmt.Errorf("%w for %s", errMissingRepositoryOwner, trimmedName)
	}
	if source.Repository == "" {
		return Dependency{}, fmt.Errorf("%w for %s", errMissingRepositoryName, trimmedName)
	}
	cleanReference := strings.TrimSpace(source.Reference)
	cleanDocPaths := sanitizeDocPaths(source.DocPaths)
	return Dependency{
		Name:      trimmedName,
		Version:   strings.TrimSpace(version),
		Ecosystem: ecosystem,
		Source: RepositorySource{
			Owner:      source.Owner,
			Repository: source.Repository,
			Reference:  cleanReference,
			DocPaths:   cleanDocPaths,
		},
	}, nil
}

func sanitizeDocPaths(paths []string) []string {
	seen := map[string]struct{}{}
	var normalized []string
	for _, path := range paths {
		trimmed := strings.Trim(strings.TrimSpace(path), "/")
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return defaultDocPaths()
	}
	return normalized
}

func defaultDocPaths() []string {
	return []string{"docs", "doc", "documentation", "README.md"}
}

// Identifier returns a stable identifier for pattern matching.
func (dependency Dependency) Identifier() string {
	return fmt.Sprintf("%s:%s", dependency.Ecosystem, dependency.Name)
}

// SafeFileName returns a deterministic file name for markdown output.
func (dependency Dependency) SafeFileName() string {
	repositoryID := fmt.Sprintf("%s-%s", dependency.Source.Owner, dependency.Source.Repository)
	normalized := sanitizeFileSegment(fmt.Sprintf("%s-%s", dependency.Ecosystem, repositoryID))
	return normalized + ".md"
}

var fileNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeFileSegment(segment string) string {
	if segment == "" {
		return "dependency"
	}
	normalized := fileNameSanitizer.ReplaceAllString(strings.ToLower(segment), "-")
	normalized = strings.Trim(normalized, "-.")
	if normalized == "" {
		return "dependency"
	}
	return normalized
}

// OutputPath resolves the target file path inside the destination directory.
func (dependency Dependency) OutputPath(baseDir string) string {
	return filepath.Join(baseDir, string(dependency.Ecosystem), dependency.SafeFileName())
}
