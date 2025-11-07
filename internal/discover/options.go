package discover

import (
	"path/filepath"
	"strings"

	"github.com/tyemirov/ctx/internal/docs/githubdoc"
)

// Options configures the discovery runner.
type Options struct {
	RootPath           string
	OutputDir          string
	Ecosystems         map[Ecosystem]bool
	IncludePatterns    []string
	ExcludePatterns    []string
	IncludeDev         bool
	IncludeIndirect    bool
	RuleSet            githubdoc.RuleSet
	Concurrency        int
	APIBase            string
	AuthorizationToken string
	NPMRegistryBase    string
	PyPIRegistryBase   string
	MaxDependencies    int
}

func (options Options) ecosystemEnabled(ecosystem Ecosystem) bool {
	if len(options.Ecosystems) == 0 {
		return true
	}
	enabled, ok := options.Ecosystems[ecosystem]
	return ok && enabled
}

func (options Options) passesFilters(dependency Dependency) bool {
	if len(options.IncludePatterns) > 0 && !matchesAnyPattern(dependency.Identifier(), options.IncludePatterns) {
		return false
	}
	if len(options.ExcludePatterns) > 0 && matchesAnyPattern(dependency.Identifier(), options.ExcludePatterns) {
		return false
	}
	return true
}

func matchesAnyPattern(value string, patterns []string) bool {
	for _, pattern := range patterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" {
			continue
		}
		if match, _ := filepath.Match(trimmed, value); match {
			return true
		}
	}
	return false
}
