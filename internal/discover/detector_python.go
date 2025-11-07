package discover

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type pythonDetector struct {
	client pypiRegistryClient
}

func newPythonDetector(client pypiRegistryClient) pythonDetector {
	if client == nil {
		client = newPyPIRegistry("")
	}
	return pythonDetector{client: client}
}

func (detector pythonDetector) Ecosystem() Ecosystem {
	return EcosystemPython
}

func (detector pythonDetector) Detect(ctx context.Context, rootPath string, options Options) ([]Dependency, error) {
	requirements, reqErr := readRequirements(filepath.Join(rootPath, "requirements.txt"))
	if reqErr != nil {
		return nil, reqErr
	}
	if options.IncludeDev {
		devRequirements, devErr := readRequirements(filepath.Join(rootPath, "requirements-dev.txt"))
		if devErr != nil {
			return nil, devErr
		}
		requirements = append(requirements, devRequirements...)
	}
	pyprojectDeps, pyprojectDev, pyprojectErr := readPyProjectDependencies(filepath.Join(rootPath, "pyproject.toml"))
	if pyprojectErr != nil {
		return nil, pyprojectErr
	}
	requirements = append(requirements, pyprojectDeps...)
	if options.IncludeDev {
		requirements = append(requirements, pyprojectDev...)
	}
	if len(requirements) == 0 {
		return nil, nil
	}
	var dependencies []Dependency
	seen := map[string]struct{}{}
	for _, requirement := range requirements {
		if _, exists := seen[requirement.Name]; exists {
			continue
		}
		metadata, metadataErr := detector.client.Metadata(ctx, requirement.Name)
		if metadataErr != nil {
			continue
		}
		owner, repository := parseGitHubRepository(metadata.ProjectURL)
		if owner == "" {
			owner, repository = parseGitHubRepository(metadata.HomePage)
		}
		if owner == "" {
			continue
		}
		source := RepositorySource{
			Owner:      owner,
			Repository: repository,
			Reference:  "main",
			DocPaths:   defaultDocPaths(),
		}
		dependency, dependencyErr := NewDependency(requirement.Name, requirement.Version, EcosystemPython, source)
		if dependencyErr != nil {
			return nil, dependencyErr
		}
		seen[requirement.Name] = struct{}{}
		dependencies = append(dependencies, dependency)
	}
	return dependencies, nil
}

type pythonRequirement struct {
	Name    string
	Version string
}

func readRequirements(path string) ([]pythonRequirement, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()
	var requirements []pythonRequirement
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "-r") || strings.HasPrefix(trimmed, "--") {
			continue
		}
		requirement := parseRequirementLine(trimmed)
		if requirement.Name != "" {
			requirements = append(requirements, requirement)
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("scan %s: %w", path, scanErr)
	}
	return requirements, nil
}

func parseRequirementLine(line string) pythonRequirement {
	clean := line
	if index := strings.Index(clean, "#"); index >= 0 {
		clean = clean[:index]
	}
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return pythonRequirement{}
	}
	name := clean
	version := ""
	separators := []string{"==", ">=", "<=", "~=", ">", "<"}
	for _, separator := range separators {
		if parts := strings.SplitN(clean, separator, 2); len(parts) == 2 {
			name = parts[0]
			version = separator + parts[1]
			break
		}
	}
	if index := strings.Index(name, "["); index >= 0 {
		name = name[:index]
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return pythonRequirement{}
	}
	return pythonRequirement{
		Name:    strings.ToLower(name),
		Version: strings.TrimSpace(version),
	}
}

func readPyProjectDependencies(path string) ([]pythonRequirement, []pythonRequirement, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()
	var deps []pythonRequirement
	var devDeps []pythonRequirement
	scanner := bufio.NewScanner(file)
	var currentArray string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[project.optional-dependencies]") {
			currentArray = ""
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentArray = ""
			continue
		}
		if strings.HasPrefix(line, "dependencies") && strings.Contains(line, "[") {
			currentArray = "dependencies"
			continue
		}
		if strings.HasPrefix(line, "dev") && strings.Contains(line, "[") {
			currentArray = "dev"
			continue
		}
		if currentArray == "" {
			continue
		}
		if line == "]" {
			currentArray = ""
			continue
		}
		trimmed := strings.Trim(line, "\",")
		requirement := parseRequirementLine(trimmed)
		if requirement.Name == "" {
			continue
		}
		if currentArray == "dependencies" {
			deps = append(deps, requirement)
			continue
		}
		if currentArray == "dev" {
			devDeps = append(devDeps, requirement)
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, nil, fmt.Errorf("scan %s: %w", path, scanErr)
	}
	return deps, devDeps, nil
}
