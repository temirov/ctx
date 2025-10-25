// Package callchain contains language-aware call chain analyzers.
package callchain

import (
	"errors"

	"github.com/tyemirov/ctx/internal/docs"
	"github.com/tyemirov/ctx/internal/types"
)

// ErrSymbolNotFound indicates that an analyzer could not locate the requested symbol.
var ErrSymbolNotFound = errors.New("callchain: symbol not found")

// AnalyzerRequest describes the input for call chain analysis.
type AnalyzerRequest struct {
	TargetSymbol            string
	MaximumDepth            int
	IncludeDocumentation    bool
	DocumentationCollector  *docs.Collector
	RepositoryRootDirectory string
}

// Analyzer resolves callers and callees for a language-specific implementation.
type Analyzer interface {
	Analyze(request AnalyzerRequest) (*types.CallChainOutput, error)
}

// Registry dispatches analysis requests across registered analyzers.
type Registry struct {
	analyzers []Analyzer
}

// NewRegistry creates a Registry for the provided analyzers.
func NewRegistry(analyzers ...Analyzer) *Registry {
	filteredAnalyzers := make([]Analyzer, 0, len(analyzers))
	for _, analyzer := range analyzers {
		if analyzer != nil {
			filteredAnalyzers = append(filteredAnalyzers, analyzer)
		}
	}
	return &Registry{analyzers: filteredAnalyzers}
}

// Analyze delegates to the first analyzer that successfully resolves the symbol.
func (registry *Registry) Analyze(request AnalyzerRequest) (*types.CallChainOutput, error) {
	for _, analyzer := range registry.analyzers {
		result, analyzeError := analyzer.Analyze(request)
		if analyzeError != nil {
			if errors.Is(analyzeError, ErrSymbolNotFound) {
				continue
			}
			return nil, analyzeError
		}
		if result != nil {
			return result, nil
		}
	}
	return nil, ErrSymbolNotFound
}
