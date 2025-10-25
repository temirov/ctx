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

var defaultCallChainRegistry = callchain.NewRegistry(
	callchain.NewGoAnalyzer(),
	callchain.NewPythonAnalyzer(),
	callchain.NewJavaScriptAnalyzer(),
)

// GetCallChainData returns call chain information for the requested symbol.
func GetCallChainData(
	targetFunctionQualifiedName string,
	callChainDepth int,
	includeDocumentation bool,
	documentationCollector *docs.Collector,
	repositoryRootDirectory string,
) (*types.CallChainOutput, error) {
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
	result, analyzeError := defaultCallChainRegistry.Analyze(request)
	if analyzeError != nil {
		if errors.Is(analyzeError, callchain.ErrSymbolNotFound) {
			return nil, fmt.Errorf(errorCallChainTargetNotFound, targetFunctionQualifiedName)
		}
		return nil, analyzeError
	}
	return result, nil
}
