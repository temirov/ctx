package docs

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/mod/modfile"

	"github.com/temirov/ctx/internal/docs/githubdoc"
	"github.com/temirov/ctx/internal/types"
)

// RemoteAttemptOptions configures remote documentation lookup for third-party modules.
type RemoteAttemptOptions struct {
	Enabled            bool
	APIBase            string
	AuthorizationToken string
}

type remoteDocumentationProvider interface {
	DocumentationForImport(importPath string) []types.DocumentationEntry
	Activate(ctx context.Context)
}

type remoteModuleMetadata struct {
	modulePath string
	owner      string
	repository string
	reference  string
	docRoots   []string
}

// RemoteDocumentationAttempt resolves documentation from GitHub for imported modules.
type RemoteDocumentationAttempt struct {
	fetcher       githubdoc.Fetcher
	modules       map[string]remoteModuleMetadata
	moduleKeys    []string
	cache         map[string][]types.DocumentationEntry
	cacheMutex    sync.RWMutex
	activeContext context.Context
}

func newRemoteDocumentationAttempt(moduleRoot string, options RemoteAttemptOptions) (*RemoteDocumentationAttempt, error) {
	if !options.Enabled {
		return nil, nil
	}
	goModPath := filepath.Join(moduleRoot, "go.mod")
	goModBytes, readError := os.ReadFile(goModPath)
	if readError != nil {
		return nil, nil
	}
	modFile, parseError := modfile.Parse("go.mod", goModBytes, nil)
	if parseError != nil {
		return nil, parseError
	}
	metadata := map[string]remoteModuleMetadata{}
	for _, require := range modFile.Require {
		modulePath := require.Mod.Path
		if !strings.HasPrefix(modulePath, "github.com/") {
			continue
		}
		segments := strings.Split(modulePath, "/")
		if len(segments) < 3 {
			continue
		}
		owner := segments[1]
		repository := segments[2]
		reference := deriveModuleReference(require.Mod.Version)
		metadata[modulePath] = remoteModuleMetadata{
			modulePath: modulePath,
			owner:      owner,
			repository: repository,
			reference:  reference,
			docRoots:   []string{"docs", "doc", "documentation"},
		}
	}
	if len(metadata) == 0 {
		return nil, nil
	}
	fetcher := githubdoc.NewFetcher(nil)
	if options.APIBase != "" {
		fetcher = fetcher.WithAPIBase(options.APIBase)
	}
	if options.AuthorizationToken != "" {
		fetcher = fetcher.WithAuthorizationToken(options.AuthorizationToken)
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		return len(keys[left]) > len(keys[right])
	})
	return &RemoteDocumentationAttempt{
		fetcher:    fetcher,
		modules:    metadata,
		moduleKeys: keys,
		cache:      map[string][]types.DocumentationEntry{},
	}, nil
}

// Activate stores the context that will be used for remote lookups.
func (attempt *RemoteDocumentationAttempt) Activate(ctx context.Context) {
	if attempt == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	attempt.cacheMutex.Lock()
	attempt.activeContext = ctx
	attempt.cacheMutex.Unlock()
}

// DocumentationForImport returns remote documentation entries for the import path.
func (attempt *RemoteDocumentationAttempt) DocumentationForImport(importPath string) []types.DocumentationEntry {
	if attempt == nil {
		return nil
	}
	modulePath := attempt.matchModule(importPath)
	if modulePath == "" {
		return nil
	}
	attempt.cacheMutex.RLock()
	if entries, ok := attempt.cache[modulePath]; ok {
		attempt.cacheMutex.RUnlock()
		return entries
	}
	attempt.cacheMutex.RUnlock()

	entries := attempt.fetchModuleDocumentation(modulePath)
	attempt.cacheMutex.Lock()
	attempt.cache[modulePath] = entries
	attempt.cacheMutex.Unlock()
	return entries
}

func (attempt *RemoteDocumentationAttempt) matchModule(importPath string) string {
	for _, key := range attempt.moduleKeys {
		if importPath == key || strings.HasPrefix(importPath, key+"/") {
			return key
		}
	}
	return ""
}

func (attempt *RemoteDocumentationAttempt) fetchModuleDocumentation(modulePath string) []types.DocumentationEntry {
	metadata, ok := attempt.modules[modulePath]
	if !ok {
		return nil
	}
	ctx := attempt.activeContext
	if ctx == nil {
		ctx = context.Background()
	}
	for _, root := range metadata.docRoots {
		documents, fetchError := attempt.fetcher.Fetch(ctx, githubdoc.FetchOptions{
			Owner:      metadata.owner,
			Repository: metadata.repository,
			Reference:  metadata.reference,
			RootPath:   root,
		})
		if fetchError != nil || len(documents) == 0 {
			continue
		}
		return convertDocumentsToEntries(modulePath, root, documents)
	}
	return nil
}

func convertDocumentsToEntries(modulePath string, root string, documents []githubdoc.Document) []types.DocumentationEntry {
	normalizedRoot := strings.Trim(strings.TrimSpace(root), "/")
	entries := make([]types.DocumentationEntry, 0, len(documents))
	for _, document := range documents {
		content := strings.TrimSpace(document.Content)
		if content == "" {
			continue
		}
		name := modulePath
		if normalizedRoot != "" {
			relative := strings.TrimPrefix(document.Path, normalizedRoot+"/")
			if relative == document.Path {
				relative = document.Path
			}
			if relative != "" {
				name = modulePath + "/" + relative
			}
		}
		entries = append(entries, types.DocumentationEntry{
			Kind: documentationKindModule,
			Name: name,
			Doc:  content,
		})
	}
	return entries
}

func deriveModuleReference(version string) string {
	if version == "" {
		return ""
	}
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return ""
	}
	if plusIndex := strings.Index(trimmed, "+"); plusIndex >= 0 {
		trimmed = trimmed[:plusIndex]
	}
	parts := strings.Split(trimmed, "-")
	if len(parts) >= 3 {
		candidate := parts[len(parts)-1]
		if len(candidate) >= 7 && isHex(candidate) {
			return candidate
		}
	}
	return trimmed
}

func isHex(value string) bool {
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}
