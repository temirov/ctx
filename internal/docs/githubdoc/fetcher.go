package githubdoc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	encodingBase64                     = "base64"
	contentTypeDirectory               = "dir"
	contentTypeFile                    = "file"
	defaultAPITimeout                  = 30 * time.Second
	defaultReference                   = "master"
	defaultAPIBaseURL                  = "https://api.github.com"
	defaultIncludeExtensionMarkdown    = ".md"
	defaultIncludeExtensionMarkdownAlt = ".mdx"
	defaultIncludeExtensionHTML        = ".html"
	defaultIncludeExtensionHTMLAlt     = ".htm"
	defaultUserAgent                   = "ctx-githubdoc-fetcher"
	headerAuthorization                = "Authorization"
	headerAccept                       = "Accept"
	headerGitHubAPIVersion             = "X-GitHub-Api-Version"
	acceptGitHubJSON                   = "application/vnd.github+json"
	githubAPIVersionValue              = "2022-11-28"
	authorizationBearerPrefix          = "Bearer "
	authorizationTokenPrefix           = "token "
)

var (
	errMissingOwner      = errors.New("repository owner is required")
	errMissingRepository = errors.New("repository name is required")
	errMissingPath       = errors.New("root path is required")
)

var (
	navigationLinkPattern = regexp.MustCompile(`\[[^\]]+\]\([^)]+\)`)
)

type httpClient interface {
	Do(request *http.Request) (*http.Response, error)
}

type apiContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	DownloadURL string `json:"download_url"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
}

type FetchOptions struct {
	Owner      string
	Repository string
	Reference  string
	RootPath   string
	RuleSet    RuleSet
}

type Document struct {
	Path    string
	Content string
}

type Fetcher struct {
	client                   httpClient
	apiBase                  string
	userAgent                string
	timeout                  time.Duration
	authorizationHeaderValue string
}

func NewFetcher(client httpClient) Fetcher {
	if client == nil {
		client = &http.Client{Timeout: defaultAPITimeout}
	}
	return Fetcher{
		client:    client,
		apiBase:   defaultAPIBaseURL,
		userAgent: defaultUserAgent,
		timeout:   defaultAPITimeout,
	}
}

func (fetcher Fetcher) WithAPIBase(base string) Fetcher {
	if base == "" {
		return fetcher
	}
	fetcher.apiBase = strings.TrimRight(base, "/")
	return fetcher
}

func (fetcher Fetcher) WithUserAgent(agent string) Fetcher {
	if agent == "" {
		return fetcher
	}
	fetcher.userAgent = agent
	return fetcher
}

func (fetcher Fetcher) WithTimeout(duration time.Duration) Fetcher {
	if duration <= 0 {
		return fetcher
	}
	fetcher.timeout = duration
	if clientWithTimeout, ok := fetcher.client.(*http.Client); ok {
		clientWithTimeout.Timeout = duration
	}
	return fetcher
}

// WithAuthorizationToken configures the fetcher to authenticate GitHub API calls.
func (fetcher Fetcher) WithAuthorizationToken(token string) Fetcher {
	fetcher.authorizationHeaderValue = formatAuthorizationHeaderValue(token)
	return fetcher
}

func (fetcher Fetcher) Fetch(ctx context.Context, options FetchOptions) ([]Document, error) {
	if options.Owner == "" {
		return nil, errMissingOwner
	}
	if options.Repository == "" {
		return nil, errMissingRepository
	}
	if strings.TrimSpace(options.RootPath) == "" {
		return nil, errMissingPath
	}

	reference := options.Reference
	if reference == "" {
		reference = defaultReference
	}
	normalizedRoot := strings.Trim(strings.TrimSpace(options.RootPath), "/")

	ruleSet := options.RuleSet.prepare()
	var documents []Document
	err := fetcher.walkDirectory(ctx, &documents, normalizedRoot, options.Owner, options.Repository, reference, ruleSet)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(documents, func(left, right int) bool {
		return documents[left].Path < documents[right].Path
	})
	return documents, nil
}

func (fetcher Fetcher) walkDirectory(ctx context.Context, accumulator *[]Document, directoryPath string, owner string, repository string, reference string, ruleSet preparedRuleSet) error {
	apiURL, buildErr := fetcher.buildContentsURL(owner, repository, directoryPath, reference)
	if buildErr != nil {
		return buildErr
	}
	payload, payloadErr := fetcher.getContent(ctx, apiURL)
	if payloadErr != nil {
		return payloadErr
	}

	for _, entry := range payload {
		if entry.Type == contentTypeDirectory {
			if ruleSet.skipDirectory != nil && ruleSet.skipDirectory.MatchString(entry.Name) {
				continue
			}
			if err := fetcher.walkDirectory(ctx, accumulator, entry.Path, owner, repository, reference, ruleSet); err != nil {
				return err
			}
			continue
		}
		if entry.Type != contentTypeFile {
			continue
		}
		extension := strings.ToLower(filepath.Ext(entry.Name))
		if !ruleSet.allowsExtension(extension) {
			continue
		}
		fileContent, fileErr := fetcher.fetchFile(ctx, entry.Path, owner, repository, reference)
		if fileErr != nil {
			return fileErr
		}
		cleanContent := ruleSet.applyTransforms(fileContent)
		*accumulator = append(*accumulator, Document{
			Path:    entry.Path,
			Content: cleanContent,
		})
	}
	return nil
}

func (fetcher Fetcher) fetchFile(ctx context.Context, filePath string, owner string, repository string, reference string) (string, error) {
	apiURL, buildErr := fetcher.buildContentsURL(owner, repository, filePath, reference)
	if buildErr != nil {
		return "", buildErr
	}
	payload, payloadErr := fetcher.getContent(ctx, apiURL)
	if payloadErr != nil {
		return "", payloadErr
	}
	if len(payload) != 1 {
		return "", fmt.Errorf("unexpected payload length for %s", filePath)
	}
	item := payload[0]
	if item.Encoding != "" && item.Encoding != encodingBase64 {
		return "", fmt.Errorf("unsupported encoding %s for %s", item.Encoding, filePath)
	}
	rawContent := item.Content
	if rawContent == "" && item.DownloadURL != "" {
		downloaded, downloadErr := fetcher.downloadFile(ctx, item.DownloadURL)
		if downloadErr != nil {
			return "", downloadErr
		}
		return downloaded, nil
	}
	if rawContent == "" {
		return "", nil
	}
	if item.Encoding == encodingBase64 {
		contentBytes, decodeErr := base64.StdEncoding.DecodeString(rawContent)
		if decodeErr != nil {
			return "", fmt.Errorf("decode content for %s: %w", filePath, decodeErr)
		}
		return string(contentBytes), nil
	}
	return rawContent, nil
}

func (fetcher Fetcher) getContent(ctx context.Context, apiURL string) ([]apiContent, error) {
	request, requestErr := fetcher.buildRequest(ctx, apiURL)
	if requestErr != nil {
		return nil, requestErr
	}
	response, responseErr := fetcher.client.Do(request)
	if responseErr != nil {
		return nil, responseErr
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 8*1024))
		return nil, fmt.Errorf("unexpected status %d for %s: %s", response.StatusCode, apiURL, string(body))
	}
	var payload interface{}
	if decodeErr := json.NewDecoder(response.Body).Decode(&payload); decodeErr != nil {
		return nil, decodeErr
	}
	switch typed := payload.(type) {
	case []interface{}:
		results := make([]apiContent, 0, len(typed))
		for _, element := range typed {
			if content, ok := convertToContent(element); ok {
				results = append(results, content)
			}
		}
		return results, nil
	case map[string]interface{}:
		if content, ok := convertToContent(typed); ok {
			return []apiContent{content}, nil
		}
	}
	return nil, fmt.Errorf("unexpected GitHub payload for %s", apiURL)
}

func (fetcher Fetcher) downloadFile(ctx context.Context, downloadURL string) (string, error) {
	request, requestErr := fetcher.buildRequest(ctx, downloadURL)
	if requestErr != nil {
		return "", requestErr
	}
	response, responseErr := fetcher.client.Do(request)
	if responseErr != nil {
		return "", responseErr
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 8*1024))
		return "", fmt.Errorf("unexpected status %d for %s: %s", response.StatusCode, downloadURL, string(body))
	}
	contentBytes, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return "", readErr
	}
	return string(contentBytes), nil
}

func (fetcher Fetcher) buildRequest(ctx context.Context, rawURL string) (*http.Request, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if requestErr != nil {
		return nil, requestErr
	}
	if fetcher.userAgent != "" {
		request.Header.Set("User-Agent", fetcher.userAgent)
	}
	if fetcher.authorizationHeaderValue != "" {
		request.Header.Set(headerAuthorization, fetcher.authorizationHeaderValue)
	}
	request.Header.Set(headerAccept, acceptGitHubJSON)
	request.Header.Set(headerGitHubAPIVersion, githubAPIVersionValue)
	return request, nil
}

func (fetcher Fetcher) buildContentsURL(owner string, repository string, itemPath string, reference string) (string, error) {
	parsedURL, parseErr := url.Parse(fetcher.apiBase)
	if parseErr != nil {
		return "", parseErr
	}
	prefix := strings.TrimSuffix(parsedURL.Path, "/")
	var builder strings.Builder
	if prefix != "" {
		builder.WriteString(prefix)
	}
	if builder.Len() == 0 {
		builder.WriteByte('/')
	}
	builder.WriteString("repos/")
	builder.WriteString(url.PathEscape(owner))
	builder.WriteByte('/')
	builder.WriteString(url.PathEscape(repository))
	builder.WriteString("/contents")
	cleanedPath := strings.Trim(strings.TrimSpace(itemPath), "/")
	if cleanedPath != "" {
		for _, segment := range strings.Split(cleanedPath, "/") {
			if segment == "" {
				continue
			}
			builder.WriteByte('/')
			builder.WriteString(url.PathEscape(segment))
		}
	}
	parsedURL.Path = builder.String()
	query := parsedURL.Query()
	if reference != "" {
		query.Set("ref", reference)
	}
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

func convertToContent(value interface{}) (apiContent, bool) {
	asMap, ok := value.(map[string]interface{})
	if !ok {
		return apiContent{}, false
	}
	content := apiContent{}
	if name, ok := asMap["name"].(string); ok {
		content.Name = name
	}
	if pathValue, ok := asMap["path"].(string); ok {
		content.Path = pathValue
	}
	if typeValue, ok := asMap["type"].(string); ok {
		content.Type = typeValue
	}
	if download, ok := asMap["download_url"].(string); ok {
		content.DownloadURL = download
	}
	if contentValue, ok := asMap["content"].(string); ok {
		content.Content = contentValue
	}
	if encodingValue, ok := asMap["encoding"].(string); ok {
		content.Encoding = encodingValue
	}
	return content, true
}

type preparedRuleSet struct {
	includeExtensions map[string]struct{}
	skipDirectory     *regexp.Regexp
	dropExpressions   []*regexp.Regexp
	stripLeadingH1    bool
	bumpHeadingsBy    int
}

func (ruleSet preparedRuleSet) allowsExtension(extension string) bool {
	if len(ruleSet.includeExtensions) == 0 {
		return true
	}
	_, ok := ruleSet.includeExtensions[extension]
	return ok
}

func (ruleSet preparedRuleSet) applyTransforms(content string) string {
	lines := strings.Split(content, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if ruleSet.shouldDrop(line) {
			continue
		}
		filtered = append(filtered, line)
	}
	if ruleSet.stripLeadingH1 {
		filtered = stripLeadingHeading(filtered)
	}
	if ruleSet.bumpHeadingsBy > 0 {
		filtered = bumpHeadings(filtered, ruleSet.bumpHeadingsBy)
	}
	filtered = removeNavigationBlocks(filtered)
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}

func (ruleSet preparedRuleSet) shouldDrop(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	for _, pattern := range ruleSet.dropExpressions {
		if pattern.MatchString(trimmed) {
			return true
		}
	}
	return false
}

func stripLeadingHeading(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	first := strings.TrimSpace(lines[0])
	if strings.HasPrefix(first, "# ") {
		return lines[1:]
	}
	return lines
}

func bumpHeadings(lines []string, increment int) []string {
	if increment <= 0 {
		return lines
	}
	prefix := strings.Repeat("#", increment)
	adjusted := make([]string, len(lines))
	for index, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if strings.HasPrefix(trimmed, "#") {
			adjusted[index] = prefix + trimmed
		} else {
			adjusted[index] = line
		}
	}
	return adjusted
}

func removeNavigationBlocks(lines []string) []string {
	result := make([]string, 0, len(lines))
	skipLinks := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			skipLinks = false
			result = append(result, line)
			continue
		}
		if looksLikeNavigationHeading(trimmed) {
			skipLinks = true
			continue
		}
		if skipLinks && isNavigationLinkLine(trimmed) {
			continue
		}
		if isNavigationLinkLine(trimmed) {
			continue
		}
		skipLinks = false
		result = append(result, line)
	}
	return result
}

func looksLikeNavigationHeading(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	lower = strings.TrimLeft(lower, "# ")
	return strings.Contains(lower, "go to") || strings.Contains(lower, "navigation")
}

func isNavigationLinkLine(line string) bool {
	matches := navigationLinkPattern.FindAllString(line, -1)
	if len(matches) < 5 {
		return false
	}
	total := 0
	for _, match := range matches {
		total += len(match)
	}
	withoutSpaces := strings.ReplaceAll(line, " ", "")
	if len(withoutSpaces) == 0 {
		return false
	}
	return float64(total)/float64(len(withoutSpaces)) >= 0.6
}

type RuleSet struct {
	IncludeExtensions []string
	SkipDirectory     string
	DropExpressions   []string
	StripLeadingH1    bool
	BumpHeadingsBy    int
}

func (ruleSet RuleSet) prepare() preparedRuleSet {
	include := ruleSet.IncludeExtensions
	if len(include) == 0 {
		include = []string{
			defaultIncludeExtensionMarkdown,
			defaultIncludeExtensionMarkdownAlt,
			defaultIncludeExtensionHTML,
			defaultIncludeExtensionHTMLAlt,
		}
	}
	includeMap := make(map[string]struct{}, len(include))
	for _, extension := range include {
		normalized := strings.ToLower(strings.TrimSpace(extension))
		if normalized == "" {
			continue
		}
		includeMap[normalized] = struct{}{}
	}
	var compiledSkip *regexp.Regexp
	if ruleSet.SkipDirectory != "" {
		compiledSkip = regexp.MustCompile(ruleSet.SkipDirectory)
	}
	var compiledDrops []*regexp.Regexp
	for _, expr := range ruleSet.DropExpressions {
		if strings.TrimSpace(expr) == "" {
			continue
		}
		compiledDrops = append(compiledDrops, regexp.MustCompile(expr))
	}
	return preparedRuleSet{
		includeExtensions: includeMap,
		skipDirectory:     compiledSkip,
		dropExpressions:   compiledDrops,
		stripLeadingH1:    ruleSet.StripLeadingH1,
		bumpHeadingsBy:    ruleSet.BumpHeadingsBy,
	}
}

func formatAuthorizationHeaderValue(rawToken string) string {
	trimmed := strings.TrimSpace(rawToken)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	bearerLower := strings.ToLower(authorizationBearerPrefix)
	tokenLower := strings.ToLower(authorizationTokenPrefix)
	if strings.HasPrefix(lower, bearerLower) || strings.HasPrefix(lower, tokenLower) {
		return trimmed
	}
	if strings.Contains(trimmed, ".") {
		return authorizationBearerPrefix + trimmed
	}
	return authorizationTokenPrefix + trimmed
}
