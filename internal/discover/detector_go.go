package discover

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

type goDetector struct{}

func (goDetector) Ecosystem() Ecosystem {
	return EcosystemGo
}

func (goDetector) Detect(ctx context.Context, rootPath string, options Options) ([]Dependency, error) {
	_ = ctx
	goModPath := filepath.Join(rootPath, "go.mod")
	bytes, readErr := os.ReadFile(goModPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return nil, nil
		}
		return nil, fmt.Errorf("read go.mod: %w", readErr)
	}
	modFile, parseErr := modfile.Parse("go.mod", bytes, nil)
	if parseErr != nil {
		return nil, fmt.Errorf("parse go.mod: %w", parseErr)
	}
	if modFile == nil {
		return nil, nil
	}
	modulePath := ""
	if modFile.Module != nil {
		modulePath = modFile.Module.Mod.Path
	}
	var dependencies []Dependency
	for _, requirement := range modFile.Require {
		if requirement == nil {
			continue
		}
		if !options.IncludeIndirect && requirement.Indirect {
			continue
		}
		if requirement.Mod.Path == "" || requirement.Mod.Path == modulePath {
			continue
		}
		source, sourceErr := goModuleSource(requirement.Mod.Path, requirement.Mod.Version)
		if sourceErr != nil {
			continue
		}
		dependency, dependencyErr := NewDependency(requirement.Mod.Path, requirement.Mod.Version, EcosystemGo, source)
		if dependencyErr != nil {
			return nil, dependencyErr
		}
		dependencies = append(dependencies, dependency)
	}
	return dependencies, nil
}

func goModuleSource(modulePath string, version string) (RepositorySource, error) {
	segments := strings.Split(modulePath, "/")
	if len(segments) < 3 {
		return RepositorySource{}, fmt.Errorf("%w: %s", errInvalidDependency, modulePath)
	}
	if segments[0] != "github.com" {
		return RepositorySource{}, fmt.Errorf("%w: %s", errUnsupportedEcosystem, modulePath)
	}
	owner := segments[1]
	repository := segments[2]
	reference := version
	if reference == "" {
		reference = "main"
	}
	return RepositorySource{
		Owner:      owner,
		Repository: repository,
		Reference:  reference,
		DocPaths:   defaultDocPaths(),
	}, nil
}
