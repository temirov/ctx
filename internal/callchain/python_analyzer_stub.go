//go:build !cgo

package callchain

// NewPythonAnalyzer returns nil when cgo is unavailable so the registry skips
// Python analysis on platforms that cannot build the tree-sitter bindings.
func NewPythonAnalyzer() Analyzer {
	return nil
}
