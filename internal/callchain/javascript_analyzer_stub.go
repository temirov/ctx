//go:build !cgo

package callchain

// NewJavaScriptAnalyzer returns nil when cgo is unavailable so the registry
// can gracefully skip JavaScript analysis on platforms that cannot build the
// tree-sitter bindings.
func NewJavaScriptAnalyzer() Analyzer {
	return nil
}
