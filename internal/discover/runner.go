package discover

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tyemirov/ctx/internal/docs/githubdoc"
)

type documentFetcher interface {
	Fetch(ctx context.Context, options githubdoc.FetchOptions) ([]githubdoc.Document, error)
}

// Runner executes dependency discovery and documentation retrieval.
type Runner struct {
	options   Options
	detectors []detector
	fetcher   documentFetcher
}

// NewRunner constructs a Runner with default dependencies.
func NewRunner(options Options) Runner {
	rootPath := options.RootPath
	if rootPath == "" {
		rootPath = "."
	}
	outputDir := options.OutputDir
	if outputDir == "" {
		outputDir = filepath.Join(rootPath, "docs", "dependencies")
	} else if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(rootPath, outputDir)
	}
	concurrency := options.Concurrency
	if concurrency <= 0 {
		concurrency = 4
	}
	if concurrency > 8 {
		concurrency = 8
	}
	npmRegistry := newNPMRegistry(options.NPMRegistryBase)
	pyPiRegistry := newPyPIRegistry(options.PyPIRegistryBase)
	fetcher := githubdoc.NewFetcher(nil).WithAPIBase(options.APIBase).WithAuthorizationToken(options.AuthorizationToken)
	return Runner{
		options: Options{
			RootPath:           rootPath,
			OutputDir:          outputDir,
			Ecosystems:         options.Ecosystems,
			IncludePatterns:    options.IncludePatterns,
			ExcludePatterns:    options.ExcludePatterns,
			IncludeDev:         options.IncludeDev,
			IncludeIndirect:    options.IncludeIndirect,
			RuleSet:            options.RuleSet,
			Concurrency:        concurrency,
			APIBase:            options.APIBase,
			AuthorizationToken: options.AuthorizationToken,
			NPMRegistryBase:    options.NPMRegistryBase,
			PyPIRegistryBase:   options.PyPIRegistryBase,
			MaxDependencies:    options.MaxDependencies,
		},
		detectors: buildDetectors(npmRegistry, pyPiRegistry),
		fetcher:   fetcher,
	}
}

// Run performs discovery and returns a manifest summary.
func (runner Runner) Run(ctx context.Context) (Summary, error) {
	dependencies, detectErr := runner.detectDependencies(ctx)
	if detectErr != nil {
		return Summary{}, detectErr
	}
	if len(dependencies) == 0 {
		return Summary{Entries: nil}, nil
	}
	entries := runner.processDependencies(ctx, dependencies)
	return Summary{Entries: entries}, nil
}

func (runner Runner) detectDependencies(ctx context.Context) ([]Dependency, error) {
	var aggregated []Dependency
	for _, detector := range runner.detectors {
		if !runner.options.ecosystemEnabled(detector.Ecosystem()) {
			continue
		}
		detected, err := detector.Detect(ctx, runner.options.RootPath, runner.options)
		if err != nil {
			return nil, err
		}
		for _, dependency := range detected {
			if !runner.options.passesFilters(dependency) {
				continue
			}
			aggregated = append(aggregated, dependency)
			if runner.options.MaxDependencies > 0 && len(aggregated) >= runner.options.MaxDependencies {
				return aggregated, nil
			}
		}
	}
	return aggregated, nil
}

func (runner Runner) processDependencies(ctx context.Context, dependencies []Dependency) []ManifestEntry {
	workerCount := runner.options.Concurrency
	if workerCount > len(dependencies) {
		workerCount = len(dependencies)
		if workerCount == 0 {
			workerCount = 1
		}
	}
	jobs := make(chan Dependency)
	results := make(chan ManifestEntry)

	var workers sync.WaitGroup
	for index := 0; index < workerCount; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for dependency := range jobs {
				results <- runner.processDependency(ctx, dependency)
			}
		}()
	}

	go func() {
		for _, dependency := range dependencies {
			jobs <- dependency
		}
		close(jobs)
		workers.Wait()
		close(results)
	}()

	var entries []ManifestEntry
	for entry := range results {
		entries = append(entries, entry)
	}
	return entries
}

func (runner Runner) processDependency(ctx context.Context, dependency Dependency) ManifestEntry {
	entry := ManifestEntry{
		Name:       dependency.Name,
		Ecosystem:  dependency.Ecosystem,
		Version:    dependency.Version,
		Repository: fmt.Sprintf("%s/%s", dependency.Source.Owner, dependency.Source.Repository),
		Reference:  dependency.Source.Reference,
		Status:     StatusSkipped,
		Reason:     "documentation not found",
	}
	docs, docErr := runner.fetchDependencyDocuments(ctx, dependency)
	if docErr != nil {
		entry.Status = StatusFailed
		entry.Reason = docErr.Error()
		return entry
	}
	if len(docs) == 0 {
		return entry
	}
	path, count, writeErr := runner.writeDocumentation(dependency, docs)
	if writeErr != nil {
		entry.Status = StatusFailed
		entry.Reason = writeErr.Error()
		return entry
	}
	entry.OutputPath = path
	entry.DocFileCount = count
	entry.Status = StatusWritten
	entry.Reason = ""
	return entry
}

func (runner Runner) fetchDependencyDocuments(ctx context.Context, dependency Dependency) ([]githubdoc.Document, error) {
	var aggregated []githubdoc.Document
	docPaths := dependency.Source.DocPaths
	if len(docPaths) == 0 {
		docPaths = defaultDocPaths()
	}
	attempted := map[string]struct{}{}
	var rootReadme string
	fetchPath := func(docPath string) error {
		normalized := strings.TrimSpace(docPath)
		if normalized == "" {
			return nil
		}
		key := strings.ToLower(normalized)
		if _, ok := attempted[key]; ok {
			return nil
		}
		attempted[key] = struct{}{}
		options := githubdoc.FetchOptions{
			Owner:      dependency.Source.Owner,
			Repository: dependency.Source.Repository,
			Reference:  dependency.Source.Reference,
			RootPath:   normalized,
			RuleSet:    runner.options.RuleSet,
		}
		documents, err := runner.fetcher.Fetch(ctx, options)
		if err != nil {
			if strings.Contains(err.Error(), "status 404") {
				return nil
			}
			return fmt.Errorf("fetch documentation for %s: %w", dependency.Name, err)
		}
		if len(documents) == 0 {
			return nil
		}
		aggregated = append(aggregated, documents...)
		if strings.EqualFold(normalized, "README.md") && rootReadme == "" {
			rootReadme = documents[0].Content
		}
		return nil
	}
	for _, docPath := range docPaths {
		if err := fetchPath(docPath); err != nil {
			return nil, err
		}
	}
	for _, hint := range extractDocPathsFromReadme(rootReadme) {
		if err := fetchPath(hint); err != nil {
			return nil, err
		}
	}
	return aggregated, nil
}

func (runner Runner) writeDocumentation(dependency Dependency, documents []githubdoc.Document) (string, int, error) {
	builder := &strings.Builder{}
	version := dependency.Version
	if version == "" {
		version = "latest"
	}
	fmt.Fprintf(builder, "# %s (%s)\n\n", dependency.Name, version)
	for _, document := range documents {
		title := filepath.Base(document.Path)
		fmt.Fprintf(builder, "## %s\n\n%s\n\n", title, strings.TrimSpace(document.Content))
	}
	outputPath := dependency.OutputPath(runner.options.OutputDir)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", 0, fmt.Errorf("create directory for %s: %w", outputPath, err)
	}
	if err := os.WriteFile(outputPath, []byte(builder.String()), 0o644); err != nil {
		return "", 0, fmt.Errorf("write %s: %w", outputPath, err)
	}
	return outputPath, len(documents), nil
}
