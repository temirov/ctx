package discover

import (
	"context"
)

type detector interface {
	Ecosystem() Ecosystem
	Detect(ctx context.Context, rootPath string, options Options) ([]Dependency, error)
}

func buildDetectors(npmClient npmRegistryClient, pypiClient pypiRegistryClient) []detector {
	return []detector{
		goDetector{},
		newJavaScriptDetector(npmClient),
		newPythonDetector(pypiClient),
	}
}
