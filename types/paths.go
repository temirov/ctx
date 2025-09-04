// Package types defines cross-package constants for file and path handling.
package types

const (
	// IgnoreFileName is the name of the file containing ignore patterns.
	IgnoreFileName = ".ignore"

	// GitIgnoreFileName is the name of the Git ignore file containing ignore patterns.
	GitIgnoreFileName = ".gitignore"

	// ExclusionPrefix marks a pattern as an exclusion directive.
	ExclusionPrefix = "EXCL:"
)
