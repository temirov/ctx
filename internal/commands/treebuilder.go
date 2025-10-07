package commands

import "github.com/temirov/ctx/internal/tokenizer"

// TreeBuilder builds directory tree nodes using configured options.
type TreeBuilder struct {
	IgnorePatterns []string
	IncludeSummary bool
	TokenCounter   tokenizer.Counter
	TokenModel     string
}
