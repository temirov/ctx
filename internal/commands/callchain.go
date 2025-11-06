package commands

import (
	"errors"
	"fmt"

	"github.com/tyemirov/ctx/internal/callchain"
	"github.com/tyemirov/ctx/internal/docs"
	"github.com/tyemirov/ctx/internal/types"
)

const (
	errorCallChainTargetNotFound = "call chain target %q not found"
)

// CallChainService exposes call chain analysis behaviour.
type CallChainService interface {
	GetCallChainData(targetFunctionQualifiedName string, callChainDepth int, includeDocumentation bool, documentationCollector *docs.Collector, repositoryRootDirectory string) (*types.CallChainOutput, error)
}

// Service implements CallChainService using a callchain.Registry.
type Service struct {
	registry *callchain.Registry
}

// NewCallChainService constructs a Service backed by the provided analyzers. When no analyzers
// are supplied the default Go, Python, and JavaScript analyzers are registered.
func NewCallChainService(analyzers ...callchain.Analyzer) *Service {
	if len(analyzers) == 0 {
		analyzers = []callchain.Analyzer{
			callchain.NewGoAnalyzer(),
			callchain.NewPythonAnalyzer(),
			callchain.NewJavaScriptAnalyzer(),
		}
	}
	return &Service{registry: callchain.NewRegistry(analyzers...)}
}

// NewCallChainServiceWithRegistry constructs a Service using the provided registry. Primarily used in tests.
func NewCallChainServiceWithRegistry(registry *callchain.Registry) *Service {
	return &Service{registry: registry}
}

// GetCallChainData returns call chain information for the requested symbol.
func (service *Service) GetCallChainData(
	targetFunctionQualifiedName string,
	callChainDepth int,
	includeDocumentation bool,
	documentationCollector *docs.Collector,
	repositoryRootDirectory string,
) (*types.CallChainOutput, error) {
	if service == nil || service.registry == nil {
		return nil, fmt.Errorf("call chain service is not configured")
	}
	if repositoryRootDirectory == "" {
		repositoryRootDirectory = "."
	}
	request := callchain.AnalyzerRequest{
		TargetSymbol:            targetFunctionQualifiedName,
		MaximumDepth:            callChainDepth,
		IncludeDocumentation:    includeDocumentation,
		DocumentationCollector:  documentationCollector,
		RepositoryRootDirectory: repositoryRootDirectory,
	}
	result, analyzeError := service.registry.Analyze(request)
	if analyzeError != nil {
		if errors.Is(analyzeError, callchain.ErrSymbolNotFound) {
			return nil, fmt.Errorf(errorCallChainTargetNotFound, targetFunctionQualifiedName)
		}
		return nil, analyzeError
	}
	return result, nil
}
