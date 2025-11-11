package webdoc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	// MinDepth is the minimum supported crawl depth for web documentation extraction.
	MinDepth = 0
	// MaxDepth is the maximum supported crawl depth for web documentation extraction.
	MaxDepth = 3

	defaultMaxPages        = 25
	maxLinksPerPage        = 40
	maxResponseBytes int64 = 2 << 20 // 2 MiB
	requestTimeout         = 20 * time.Second
	userAgentValue         = "ctx-webdoc-fetcher"
)

var (
	errMissingURL = errors.New("path is required")

	commentPattern  = regexp.MustCompile(`(?s)<!--.*?-->`)
	scriptPattern   = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)
	stylePattern    = regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style>`)
	anchorPattern   = regexp.MustCompile(`(?is)<a\b[^>]*?href\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>` + "`" + `]+))`)
	hrefAttrPattern = regexp.MustCompile(`(?is)href\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>` + "`" + `]+))`)
)

type httpClient interface {
	Do(request *http.Request) (*http.Response, error)
}

// Page represents a sanitized documentation page collected from the web.
type Page struct {
	URL     string
	Title   string
	Content string
}

// Fetcher crawls the web for documentation pages.
type Fetcher struct {
	client   httpClient
	maxPages int
}

// NewFetcher returns a Fetcher backed by the provided HTTP client or a default client when nil.
func NewFetcher(client httpClient) Fetcher {
	if client == nil {
		client = &http.Client{Timeout: requestTimeout}
	}
	return Fetcher{
		client:   client,
		maxPages: defaultMaxPages,
	}
}

// Fetch retrieves sanitized documentation pages up to the requested depth starting from rootURL.
func (fetcher Fetcher) Fetch(ctx context.Context, rootURL string, depth int) ([]Page, error) {
	trimmed := strings.TrimSpace(rootURL)
	if trimmed == "" {
		return nil, errMissingURL
	}
	if depth < MinDepth {
		return nil, fmt.Errorf("depth must be >= %d", MinDepth)
	}
	if depth > MaxDepth {
		return nil, fmt.Errorf("depth must be <= %d", MaxDepth)
	}
	parsedRoot, parseErr := url.Parse(trimmed)
	if parseErr != nil {
		return nil, fmt.Errorf("parse url: %w", parseErr)
	}
	if !isSupportedScheme(parsedRoot.Scheme) {
		return nil, fmt.Errorf("unsupported url scheme %s", parsedRoot.Scheme)
	}
	normalizeURL(parsedRoot)
	type crawlTarget struct {
		url   *url.URL
		depth int
	}
	queue := []crawlTarget{{url: parsedRoot, depth: 0}}
	pending := map[string]struct{}{canonicalURL(parsedRoot): {}}
	visited := map[string]struct{}{}
	var pages []Page
	for len(queue) > 0 && len(pages) < fetcher.maxPages {
		current := queue[0]
		queue = queue[1:]
		currentKey := canonicalURL(current.url)
		delete(pending, currentKey)
		if _, seen := visited[currentKey]; seen {
			continue
		}
		page, links, fetchErr := fetcher.retrievePage(ctx, current.url)
		if fetchErr != nil {
			return nil, fmt.Errorf("fetch %s: %w", current.url.String(), fetchErr)
		}
		visited[currentKey] = struct{}{}
		if page != nil {
			pages = append(pages, *page)
		}
		if current.depth >= depth {
			continue
		}
		for _, link := range links {
			normalized := fetcher.normalizeLink(parsedRoot, link)
			if normalized == nil {
				continue
			}
			linkKey := canonicalURL(normalized)
			if _, seen := visited[linkKey]; seen {
				continue
			}
			if _, queued := pending[linkKey]; queued {
				continue
			}
			queue = append(queue, crawlTarget{url: normalized, depth: current.depth + 1})
			pending[linkKey] = struct{}{}
			if len(queue)+len(pages) >= fetcher.maxPages {
				break
			}
		}
	}
	return pages, nil
}

func (fetcher Fetcher) retrievePage(ctx context.Context, target *url.URL) (*Page, []*url.URL, error) {
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if requestErr != nil {
		return nil, nil, fmt.Errorf("build request: %w", requestErr)
	}
	request.Header.Set("User-Agent", userAgentValue)
	response, err := fetcher.client.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, nil, fmt.Errorf("unexpected status %d", response.StatusCode)
	}
	if contentType := response.Header.Get("Content-Type"); contentType != "" && !isHTMLContentType(contentType) {
		return nil, nil, fmt.Errorf("unsupported content type %s", contentType)
	}
	limitedReader := io.LimitReader(response.Body, maxResponseBytes)
	rawBytes, readErr := io.ReadAll(limitedReader)
	if readErr != nil {
		return nil, nil, fmt.Errorf("read response: %w", readErr)
	}
	raw := string(rawBytes)
	cleaned := stripNoise(raw)
	links := extractAnchorLinks(cleaned, maxLinksPerPage)
	content, extractedTitle := renderSanitizedContent(cleaned)
	title := strings.TrimSpace(extractedTitle)
	if title == "" {
		title = target.String()
	}
	return &Page{
		URL:     canonicalURL(target),
		Title:   title,
		Content: content,
	}, links, nil
}

func (fetcher Fetcher) normalizeLink(root *url.URL, candidate *url.URL) *url.URL {
	if candidate == nil {
		return nil
	}
	resolved := candidate
	if !candidate.IsAbs() {
		resolved = root.ResolveReference(candidate)
	}
	if !isSupportedScheme(resolved.Scheme) {
		return nil
	}
	if !strings.EqualFold(resolved.Host, root.Host) {
		return nil
	}
	clean := cloneURL(resolved)
	normalizeURL(clean)
	return clean
}

func canonicalURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	copy := cloneURL(u)
	normalizeURL(copy)
	return copy.String()
}

func normalizeURL(u *url.URL) {
	if u == nil {
		return
	}
	u.Fragment = ""
	if u.Path == "" {
		u.Path = "/"
		return
	}
	cleaned := path.Clean(u.Path)
	if cleaned == "." {
		cleaned = "/"
	}
	u.Path = cleaned
}

func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	copy := *u
	return &copy
}

func isSupportedScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}

func isHTMLContentType(header string) bool {
	lower := strings.ToLower(header)
	return strings.Contains(lower, "text/html") || strings.Contains(lower, "application/xhtml+xml")
}

func stripNoise(input string) string {
	withoutScripts := scriptPattern.ReplaceAllString(input, " ")
	withoutStyles := stylePattern.ReplaceAllString(withoutScripts, " ")
	return commentPattern.ReplaceAllString(withoutStyles, " ")
}

func extractAnchorLinks(input string, limit int) []*url.URL {
	if limit <= 0 {
		return nil
	}
	var results []*url.URL
	seen := map[string]struct{}{}
	matches := anchorPattern.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		var candidate string
		for i := 1; i < len(match); i++ {
			if match[i] != "" {
				candidate = match[i]
				break
			}
		}
		candidate = strings.TrimSpace(html.UnescapeString(candidate))
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		parsed, err := url.Parse(candidate)
		if err != nil {
			continue
		}
		results = append(results, parsed)
		seen[candidate] = struct{}{}
		if len(results) >= limit {
			break
		}
	}
	return results
}

func renderSanitizedContent(input string) (string, string) {
	var builder strings.Builder
	state := &renderState{}
	var tagBuffer bytes.Buffer
	inTag := false
	for i := 0; i < len(input); i++ {
		ch := input[i]
		if inTag {
			if ch == '>' {
				processTag(tagBuffer.String(), &builder, state)
				tagBuffer.Reset()
				inTag = false
				continue
			}
			tagBuffer.WriteByte(ch)
			continue
		}
		if ch == '<' {
			inTag = true
			continue
		}
		state.writeChar(&builder, ch)
	}
	content := collapseBlankLines(html.UnescapeString(builder.String()))
	title := strings.TrimSpace(html.UnescapeString(state.titleBuilder.String()))
	return content, title
}

type renderState struct {
	listDepth    int
	inPre        bool
	inCode       bool
	inTitle      bool
	inLink       bool
	linkHref     string
	titleBuilder strings.Builder
}

func (state *renderState) writeChar(builder *strings.Builder, ch byte) {
	if state.inTitle {
		state.titleBuilder.WriteByte(ch)
		return
	}
	if state.inPre {
		builder.WriteByte(ch)
		return
	}
	if unicode.IsSpace(rune(ch)) {
		builder.WriteByte(' ')
		return
	}
	builder.WriteByte(ch)
}

func processTag(raw string, builder *strings.Builder, state *renderState) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}
	closing := false
	if strings.HasPrefix(raw, "/") {
		closing = true
		raw = strings.TrimSpace(raw[1:])
	}
	selfClosing := strings.HasSuffix(raw, "/")
	if selfClosing {
		raw = strings.TrimSpace(strings.TrimSuffix(raw, "/"))
	}
	name, attrs := splitTag(raw)
	switch name {
	case "title":
		state.inTitle = !closing
	case "br":
		builder.WriteString("\n")
	case "p", "div", "section", "article", "main":
		if !closing {
			builder.WriteString("\n\n")
		}
	case "h1", "h2", "h3", "h4", "h5", "h6":
		if closing {
			builder.WriteString("\n")
			return
		}
		builder.WriteString("\n\n")
		builder.WriteString(strings.Repeat("#", headingLevel(name)))
		builder.WriteByte(' ')
	case "ul", "ol":
		if closing {
			if state.listDepth > 0 {
				state.listDepth--
			}
			builder.WriteString("\n")
		} else {
			state.listDepth++
		}
	case "li":
		if closing {
			return
		}
		builder.WriteByte('\n')
		if state.listDepth > 0 {
			builder.WriteString(strings.Repeat("  ", state.listDepth-1))
		}
		builder.WriteString("- ")
	case "pre":
		if closing {
			if state.inPre {
				builder.WriteString("\n```\n")
				state.inPre = false
			}
		} else if !state.inPre {
			builder.WriteString("\n\n```\n")
			state.inPre = true
		}
	case "code":
		if closing {
			if state.inCode {
				builder.WriteByte('`')
				state.inCode = false
			}
		} else if !state.inCode {
			builder.WriteByte('`')
			state.inCode = true
		}
	case "a":
		if closing {
			if state.inLink && state.linkHref != "" {
				builder.WriteString(" (")
				builder.WriteString(state.linkHref)
				builder.WriteString(")")
			}
			state.inLink = false
			state.linkHref = ""
		} else {
			state.inLink = true
			state.linkHref = extractHref(attrs)
		}
	}
}

func splitTag(raw string) (string, string) {
	if raw == "" {
		return "", ""
	}
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return "", ""
	}
	name := strings.ToLower(parts[0])
	attrs := strings.TrimPrefix(raw, parts[0])
	return name, attrs
}

func extractHref(raw string) string {
	if raw == "" {
		return ""
	}
	match := hrefAttrPattern.FindStringSubmatch(raw)
	if len(match) == 0 {
		return ""
	}
	for i := 1; i < len(match); i++ {
		if match[i] != "" {
			return strings.TrimSpace(html.UnescapeString(match[i]))
		}
	}
	return ""
}

func headingLevel(tag string) int {
	switch tag {
	case "h1":
		return 1
	case "h2":
		return 2
	case "h3":
		return 3
	case "h4":
		return 4
	case "h5":
		return 5
	default:
		return 6
	}
}

func collapseBlankLines(value string) string {
	lines := strings.Split(value, "\n")
	var cleaned []string
	blankPending := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if !blankPending {
				cleaned = append(cleaned, "")
				blankPending = true
			}
			continue
		}
		blankPending = false
		cleaned = append(cleaned, trimmed)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}
