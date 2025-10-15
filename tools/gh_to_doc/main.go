package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ---------- GitHub API types ----------

type githubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"` // "file" or "dir"
	DownloadURL string `json:"download_url"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
}

type docItem struct {
	RelPath string
	Lines   []string
	Title   string
}

// ---------- rules (YAML/JSON) ----------

// boundaryRule is the human-friendly way to define a top/bottom block to remove.
// NOTE (per your semantics):
// - "footer" applies to the TOP of the file (think: breadcrumb/links block)
// - "header" applies to the BOTTOM of the file (think: repeated footer links)
type boundaryRule struct {
	WithinFirst  *int     `yaml:"within_first"   json:"within_first"`   // scan the first N lines (for "footer" at top)
	WithinLast   *int     `yaml:"within_last"    json:"within_last"`    // scan the last N lines (for "header" at bottom)
	StartsWith   []string `yaml:"starts_with"    json:"starts_with"`    // match if a line starts with any of these
	ContainsAny  []string `yaml:"contains_any"   json:"contains_any"`   // or contains any of these
	DropUntil    string   `yaml:"drop_until"     json:"drop_until"`     // "blank_line" | "previous_blank_line" | ""(max only)
	MaxDropLines *int     `yaml:"max_drop_lines" json:"max_drop_lines"` // safety cap (default = window size)
}

type rulesFile struct {
	Name             string   `yaml:"name" json:"name"`
	RepoURL          string   `yaml:"repo_url" json:"repo_url"`
	PathRoot         string   `yaml:"path_root" json:"path_root"`
	SkipDirPattern   string   `yaml:"skip_dir_pattern" json:"skip_dir_pattern"`
	IncludeExts      []string `yaml:"include_exts" json:"include_exts"`
	GenericHeadingRe string   `yaml:"generic_heading_regex" json:"generic_heading_regex"`
	DropLineRegexps  []string `yaml:"drop_line_regexps" json:"drop_line_regexps"`

	// Human-friendly block rules:
	// "footer" = top block, "header" = bottom block (per your wording)
	Footer *boundaryRule `yaml:"footer" json:"footer"`
	Header *boundaryRule `yaml:"header" json:"header"`

	TOC struct {
		MaxLines       *int     `yaml:"max_lines" json:"max_lines"`
		MinLinkDensity *float64 `yaml:"min_link_density" json:"min_link_density"`
	} `yaml:"toc" json:"toc"`

	Content struct {
		StripLeadingH1 *bool `yaml:"strip_leading_h1" json:"strip_leading_h1"`
		BumpHeadingsBy *int  `yaml:"bump_headings_by" json:"bump_headings_by"`
		RewriteMDLinks *bool `yaml:"rewrite_md_links" json:"rewrite_md_links"`
	} `yaml:"content" json:"content"`

	Post struct {
		DedupeTitles             *bool    `yaml:"dedupe_titles" json:"dedupe_titles"`
		CommonHeadTailQuorum     *float64 `yaml:"common_head_tail_quorum" json:"common_head_tail_quorum"`
		CommonHeadTailProbeLines *int     `yaml:"common_head_tail_probe_lines" json:"common_head_tail_probe_lines"`
	} `yaml:"post" json:"post"`
}

func loadRulesFile(pathToFile string) (*rulesFile, error) {
	b, err := os.ReadFile(pathToFile)
	if err != nil {
		return nil, err
	}
	rf := new(rulesFile)
	// Try YAML first
	if err := yaml.Unmarshal(b, rf); err == nil && (rf.Name != "" || rf.RepoURL != "" || len(rf.DropLineRegexps) > 0 || rf.Footer != nil || rf.Header != nil) {
		return rf, nil
	}
	// Fallback to JSON
	if err := json.Unmarshal(b, rf); err != nil {
		return nil, fmt.Errorf("cannot parse %s as YAML or JSON: %w", pathToFile, err)
	}
	return rf, nil
}

// ---------- regex helpers ----------

var (
	mdHeadingRE   = regexp.MustCompile(`^\s*#{1,6}\s+(.+)$`)
	mdLinkRE      = regexp.MustCompile(`\[((?:\\\]|[^\]])*)\]\(([^)]+)\)`)
	mdImageRE     = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	htmlHeaderRE  = regexp.MustCompile(`(?is)<header\b.*?</header>`)
	htmlNavRE     = regexp.MustCompile(`(?is)<nav\b.*?</nav>`)
	htmlFooterRE  = regexp.MustCompile(`(?is)<footer\b.*?</footer>`)
	htmlScriptRE  = regexp.MustCompile(`(?is)<script\b.*?</script>`)
	htmlStyleRE   = regexp.MustCompile(`(?is)<style\b.*?</style>`)
	multiBlankRE  = regexp.MustCompile(`\n{3,}`)
	yamlFrontRE   = regexp.MustCompile(`(?s)^\s*---\s*\n.*?\n---\s*\n`)
	nonAnchorChar = regexp.MustCompile(`[^a-z0-9\s-]`)

	// For quick "linky" TOC detection
	topTOCLinkLike = regexp.MustCompile(`\[[^\]]+\]\([^)]+\)`)
)

// ---------- main ----------

func main() {
	var repositoryURL, owner, repo, ref, docsRoot, outPath, rulesPath string
	var includeCSV string
	var includeTOC, fileLabels bool
	var stripLeadingH1, bumpHeadings, rewriteLinks, dedupeTitles bool
	var genericHeadingReStr, skipDirPattern string
	var quorum float64
	var probeLines int
	var tocMaxLines int
	var tocMinLinkDensity float64
	var verbose bool

	flag.StringVar(&repositoryURL, "repo-url", "", "GitHub URL like https://github.com/org/repo/tree/branch/path")
	flag.StringVar(&owner, "owner", "", "Repo owner (overrides repo-url)")
	flag.StringVar(&repo, "repo", "", "Repo name (overrides repo-url)")
	flag.StringVar(&ref, "ref", "master", "Git ref (branch, tag, SHA)")
	flag.StringVar(&docsRoot, "path", "", "Path inside repo (overrides repo-url)")
	flag.StringVar(&outPath, "out", "docs-clean.md", "Output Markdown file")
	flag.StringVar(&rulesPath, "rules", "", "YAML/JSON rules file")
	flag.StringVar(&includeCSV, "exts", ".md,.mdx,.html,.htm", "Comma-separated extensions")
	flag.BoolVar(&includeTOC, "toc", true, "Include table of contents")
	flag.BoolVar(&fileLabels, "file-labels", false, "Show source file path before content")
	flag.BoolVar(&stripLeadingH1, "strip-leading-h1", true, "Strip the first H1 per file")
	flag.BoolVar(&bumpHeadings, "bump-headings", true, "Increase heading levels by +1")
	flag.BoolVar(&rewriteLinks, "rewrite-links", true, "Rewrite relative links to GitHub blob URLs")
	flag.BoolVar(&dedupeTitles, "dedupe-titles", true, "Drop duplicate section titles")
	flag.StringVar(&genericHeadingReStr, "generic-heading-regex", `(?i)^(element|overview|introduction|readme|guide|documentation)$`, "Ignore these headings as titles")
	flag.StringVar(&skipDirPattern, "skip-dir-pattern", `(?i)^(v\d+|version[-_ ]?\d+|legacy|old|archive|archived)$`, "Skip matching directories")
	flag.Float64Var(&quorum, "common-quorum", 0.70, "Quorum for common head/tail trimming")
	flag.IntVar(&probeLines, "common-probe", 80, "Probe lines for common head/tail")
	flag.IntVar(&tocMaxLines, "toc-max-lines", 50, "Top lines to check for a link-dense TOC")
	flag.Float64Var(&tocMinLinkDensity, "toc-min-density", 0.45, "Min link density to strip a top TOC")
	flag.BoolVar(&verbose, "v", false, "Verbose logs")
	flag.Parse()

	// Parse repo-url if provided
	pOwner, pRepo, pRef, pPath := parseRepoURL(repositoryURL)
	if owner == "" {
		owner = pOwner
	}
	if repo == "" {
		repo = pRepo
	}
	if docsRoot == "" {
		docsRoot = pPath
	}
	if ref == "master" && pRef != "" {
		ref = pRef
	}
	if owner == "" || repo == "" || docsRoot == "" {
		fail(errors.New("owner/repo/path are required (use --repo-url or --owner --repo --path)"))
	}

	// include extensions set
	includeExts := parseCSVSet(includeCSV)

	// Load rules if present
	var rules *rulesFile
	var compiledDropRules []*regexp.Regexp
	if rulesPath != "" {
		rf, err := loadRulesFile(rulesPath)
		if err != nil {
			fail(err)
		}
		rules = rf

		// allow rules to override knobs (but CLI flags still respected as current values)
		if rf.PathRoot != "" && docsRoot == "" {
			docsRoot = rf.PathRoot
		}
		if rf.SkipDirPattern != "" {
			skipDirPattern = rf.SkipDirPattern
		}
		if len(rf.IncludeExts) > 0 {
			includeExts = parseCSVSet(strings.Join(rf.IncludeExts, ","))
		}
		if rf.GenericHeadingRe != "" {
			genericHeadingReStr = rf.GenericHeadingRe
		}
		if len(rf.DropLineRegexps) > 0 {
			for _, p := range rf.DropLineRegexps {
				compiledDropRules = append(compiledDropRules, regexp.MustCompile(p))
			}
		}
		if rf.TOC.MaxLines != nil {
			tocMaxLines = *rf.TOC.MaxLines
		}
		if rf.TOC.MinLinkDensity != nil {
			tocMinLinkDensity = *rf.TOC.MinLinkDensity
		}
		if rf.Content.StripLeadingH1 != nil {
			stripLeadingH1 = *rf.Content.StripLeadingH1
		}
		if rf.Content.BumpHeadingsBy != nil {
			bumpHeadings = *rf.Content.BumpHeadingsBy > 0
		}
		if rf.Content.RewriteMDLinks != nil {
			rewriteLinks = *rf.Content.RewriteMDLinks
		}
		if rf.Post.DedupeTitles != nil {
			dedupeTitles = *rf.Post.DedupeTitles
		}
		if rf.Post.CommonHeadTailQuorum != nil {
			quorum = *rf.Post.CommonHeadTailQuorum
		}
		if rf.Post.CommonHeadTailProbeLines != nil {
			probeLines = *rf.Post.CommonHeadTailProbeLines
		}
	}
	if len(compiledDropRules) == 0 {
		compiledDropRules = defaultDropLineRules()
	}

	compiledSkipDir := regexp.MustCompile(skipDirPattern)
	genericHeadingRe := regexp.MustCompile(genericHeadingReStr)

	// fetch & build
	ctx := context.Background()
	client := &http.Client{Timeout: 30 * time.Second}
	token := os.Getenv("GITHUB_TOKEN")

	items, err := walkAndLoad(ctx, client, token, owner, repo, ref, docsRoot, includeExts, compiledSkipDir, verbose)
	if err != nil {
		fail(err)
	}
	if len(items) == 0 {
		fail(errors.New("no documents found"))
	}

	for i := range items {
		// Apply human-friendly header/footer removal first (uses YAML rules)
		if rules != nil {
			items[i].Lines = applyHeaderFooter(items[i].Lines, rules)
		}

		// Optional link rewrite (make relative links/images absolute to GitHub blob URLs)
		if rewriteLinks {
			items[i].Lines = rewriteRelativeLinks(items[i].Lines, owner, repo, ref, path.Dir(items[i].RelPath))
		}

		// Strip a top link-dense TOC if present
		items[i].Lines = stripTopTOC(items[i].Lines, tocMaxLines, tocMinLinkDensity)

		// Drop boilerplate lines via regex rules
		items[i].Lines = sanitizeLines(items[i].Lines, compiledDropRules)

		if stripLeadingH1 {
			items[i].Lines = stripFirstH1(items[i].Lines)
		}
		if bumpHeadings {
			items[i].Lines = bumpHeadingLevels(items[i].Lines, 1)
		}
		items[i].Title = firstHeadingOrPathTitle(items[i].Lines, items[i].RelPath, genericHeadingRe)
	}

	if dedupeTitles {
		items = dedupeByTitle(items)
	}

	headTrim := commonRunLenStart(items, quorum, probeLines)
	tailTrim := commonRunLenEnd(items, quorum, probeLines)

	var b strings.Builder
	b.WriteString("# Consolidated Documentation\n\n")
	b.WriteString(fmt.Sprintf("_Source: %s/%s • Path: %s • Ref: %s • Generated: %s_\n\n",
		owner, repo, docsRoot, ref, time.Now().Format(time.RFC3339)))

	if includeTOC {
		b.WriteString("## Table of Contents\n\n")
		for _, it := range items {
			anc := anchorFromTitle(it.Title)
			b.WriteString(fmt.Sprintf("- [%s](#%s)\n", it.Title, anc))
		}
		b.WriteString("\n---\n\n")
	}

	for _, it := range items {
		content := sliceWithTrim(it.Lines, headTrim, tailTrim)
		if !hasNonEmpty(content) {
			continue
		}
		anc := anchorFromTitle(it.Title)
		b.WriteString(fmt.Sprintf("## %s {#%s}\n\n", it.Title, anc))
		if fileLabels {
			b.WriteString(fmt.Sprintf("_%s_\n\n", it.RelPath))
		}
		b.WriteString(strings.Join(content, "\n"))
		if !strings.HasSuffix(b.String(), "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n---\n\n")
	}

	if err := os.WriteFile(outPath, []byte(b.String()), 0o644); err != nil {
		fail(err)
	}
	if verbose {
		fmt.Printf("Wrote %s • files:%d • trimmed head:%d tail:%d\n", outPath, len(items), headTrim, tailTrim)
	}
}

// ---------- fetch & crawl ----------

func walkAndLoad(ctx context.Context, httpClient *http.Client, token, owner, repo, ref, startPath string, includeExts map[string]struct{}, skipDir *regexp.Regexp, verbose bool) ([]docItem, error) {
	var out []docItem
	queue := []string{startPath}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		ents, err := listDirectory(ctx, httpClient, token, owner, repo, ref, cur)
		if err != nil {
			return nil, fmt.Errorf("list %s: %w", cur, err)
		}
		sort.Slice(ents, func(i, j int) bool { return strings.ToLower(ents[i].Name) < strings.ToLower(ents[j].Name) })
		for _, e := range ents {
			switch e.Type {
			case "dir":
				if skipDir.MatchString(e.Name) {
					if verbose {
						fmt.Printf("skip dir: %s\n", e.Path)
					}
					continue
				}
				queue = append(queue, e.Path)
			case "file":
				ext := strings.ToLower(path.Ext(e.Name))
				if _, ok := includeExts[ext]; !ok {
					continue
				}
				raw, err := fetchFile(ctx, httpClient, token, owner, repo, ref, e.Path)
				if err != nil {
					return nil, fmt.Errorf("fetch %s: %w", e.Path, err)
				}
				lines := normalizeAndSplit(raw, ext)
				out = append(out, docItem{RelPath: e.Path, Lines: lines})
				if verbose {
					fmt.Printf("add: %s (%d lines)\n", e.Path, len(lines))
				}
			}
		}
	}
	return out, nil
}

func listDirectory(ctx context.Context, httpClient *http.Client, token, owner, repo, ref, dirPath string) ([]githubContent, error) {
	api := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(dirPath))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	q := req.URL.Query()
	q.Set("ref", ref)
	req.URL.RawQuery = q.Encode()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list %s: status %d: %s", dirPath, resp.StatusCode, string(body))
	}
	var items []githubContent
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func fetchFile(ctx context.Context, httpClient *http.Client, token, owner, repo, ref, filePath string) (string, error) {
	api := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(filePath))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	q := req.URL.Query()
	q.Set("ref", ref)
	req.URL.RawQuery = q.Encode()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("fetch %s: status %d: %s", filePath, resp.StatusCode, string(body))
	}
	var gc githubContent
	if err := json.NewDecoder(resp.Body).Decode(&gc); err != nil {
		return "", err
	}
	if gc.Encoding == "base64" {
		dec, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(gc.Content, "\n", ""))
		if err != nil {
			return "", err
		}
		return string(dec), nil
	}
	if gc.DownloadURL != "" {
		req2, _ := http.NewRequestWithContext(ctx, http.MethodGet, gc.DownloadURL, nil)
		resp2, err2 := httpClient.Do(req2)
		if err2 != nil {
			return "", err2
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != 200 {
			body, _ := io.ReadAll(resp2.Body)
			return "", fmt.Errorf("raw %s: status %d: %s", filePath, resp2.StatusCode, string(body))
		}
		b, _ := io.ReadAll(resp2.Body)
		return string(b), nil
	}
	return "", fmt.Errorf("unsupported encoding for %s", filePath)
}

// ---------- normalization & cleanup ----------

func normalizeAndSplit(raw, ext string) []string {
	text := strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", "\n"), "\r", "\n")
	if ext == ".md" || ext == ".mdx" {
		text = yamlFrontRE.ReplaceAllString(text, "")
	}
	if ext == ".html" || ext == ".htm" {
		text = htmlHeaderRE.ReplaceAllString(text, "")
		text = htmlNavRE.ReplaceAllString(text, "")
		text = htmlFooterRE.ReplaceAllString(text, "")
		text = htmlScriptRE.ReplaceAllString(text, "")
		text = htmlStyleRE.ReplaceAllString(text, "")
	}
	text = multiBlankRE.ReplaceAllString(text, "\n\n")
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return lines
}

func sanitizeLines(lines []string, dropRules []*regexp.Regexp) []string {
	var kept []string
	for _, ln := range lines {
		ts := strings.TrimSpace(ln)
		if ts == "" {
			kept = append(kept, ln)
			continue
		}
		drop := false
		for _, re := range dropRules {
			if re.MatchString(ts) {
				drop = true
				break
			}
		}
		if !drop {
			kept = append(kept, ln)
		}
	}
	return collapseBlankRuns(kept)
}

func defaultDropLineRules() []*regexp.Regexp {
	pats := []string{
		`(?i)^\s*\[?edit (this )?page\]?\(.*github.*\)\s*$`,
		`(?i)^edit this page( on github)?`,
		`(?i)^back to top$`,
		`(?i)^\s*copyright\b.*$`,
		`(?i)^\s*license\b.*$`,
		`(?i)^\s*subscribe to our newsletter.*$`,
		`(?i)^\s*products\s*$`,
		`(?i)^\s*enterprise extensions\s*$`,
		`(?i)^\s*—\s*$`,
		`(?i)^\s*##\s*go to\s*$`,
	}
	var res []*regexp.Regexp
	for _, p := range pats {
		res = append(res, regexp.MustCompile(p))
	}
	return res
}

func stripTopTOC(lines []string, maxProbe int, minLinkDensity float64) []string {
	if len(lines) == 0 || maxProbe <= 0 {
		return lines
	}
	end := maxProbe
	if end > len(lines) {
		end = len(lines)
	}
	seg := lines[:end]

	linky := 0
	nonEmpty := 0
	for _, ln := range seg {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		nonEmpty++
		if topTOCLinkLike.MatchString(t) || strings.HasPrefix(t, "- [") || strings.HasPrefix(t, "* [") {
			linky++
		}
	}
	if nonEmpty == 0 {
		return lines
	}
	if float64(linky)/float64(nonEmpty) < minLinkDensity {
		return lines
	}
	// cut until first blank line after the linky area
	cut := 0
	for i, ln := range seg {
		if strings.TrimSpace(ln) == "" && i > 0 {
			cut = i + 1
			break
		}
	}
	if cut == 0 {
		cut = end
	}
	return lines[cut:]
}

func stripFirstH1(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	if mdHeadingRE.MatchString(lines[0]) && strings.HasPrefix(strings.TrimSpace(lines[0]), "# ") {
		out := append([]string{}, lines[1:]...)
		if len(out) > 0 && strings.TrimSpace(out[0]) == "" {
			return out[1:]
		}
		return out
	}
	return lines
}

func bumpHeadingLevels(lines []string, delta int) []string {
	out := make([]string, len(lines))
	for i, ln := range lines {
		trim := strings.TrimLeft(ln, " ")
		if !strings.HasPrefix(trim, "#") {
			out[i] = ln
			continue
		}
		count := 0
		for _, ch := range trim {
			if ch == '#' {
				count++
			} else {
				break
			}
		}
		if count == 0 {
			out[i] = ln
			continue
		}
		newCount := count + delta
		if newCount > 6 {
			newCount = 6
		}
		rest := strings.TrimSpace(trim[count:])
		out[i] = strings.Repeat("#", newCount) + " " + rest
	}
	return out
}

func collapseBlankRuns(lines []string) []string {
	var out []string
	lastBlank := false
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			if !lastBlank {
				out = append(out, "")
			}
			lastBlank = true
		} else {
			out = append(out, ln)
			lastBlank = false
		}
	}
	return out
}

// ---------- header/footer application ----------

func applyHeaderFooter(lines []string, rf *rulesFile) []string {
	if rf == nil {
		return lines
	}
	// "footer" (per your semantics) = block at TOP
	if rf.Footer != nil {
		lines = dropBlockTop(lines, rf.Footer)
	}
	// "header" (per your semantics) = block at BOTTOM
	if rf.Header != nil {
		lines = dropBlockBottom(lines, rf.Header)
	}
	return lines
}

func dropBlockTop(lines []string, br *boundaryRule) []string {
	n := len(lines)
	if n == 0 {
		return lines
	}
	window := getInt(br.WithinFirst, 0)
	if window <= 0 {
		return lines
	}
	if window > n {
		window = n
	}
	maxDrop := getInt(br.MaxDropLines, window)

	startIdx := -1
	for i := 0; i < window; i++ {
		if matchesBoundaryLine(lines[i], br) {
			startIdx = i
			break
		}
	}
	if startIdx < 0 {
		return lines
	}

	endIdx := startIdx + 1
	switch strings.ToLower(strings.TrimSpace(br.DropUntil)) {
	case "blank_line":
		for endIdx < minInt(n, startIdx+maxDrop) {
			if strings.TrimSpace(lines[endIdx]) == "" {
				endIdx++ // include the blank
				break
			}
			endIdx++
		}
	default:
		// cap by maxDrop only
		endIdx = minInt(n, startIdx+maxDrop)
	}
	return lines[endIdx:]
}

func dropBlockBottom(lines []string, br *boundaryRule) []string {
	n := len(lines)
	if n == 0 {
		return lines
	}
	window := getInt(br.WithinLast, 0)
	if window <= 0 {
		return lines
	}
	if window > n {
		window = n
	}
	maxDrop := getInt(br.MaxDropLines, window)

	// scan upwards inside the bottom window to find the FIRST matching line from bottom
	start := n - window
	hit := -1
	for i := n - 1; i >= start; i-- {
		if matchesBoundaryLine(lines[i], br) {
			hit = i
			break
		}
	}
	if hit < 0 {
		return lines
	}

	// drop from previous blank line before 'hit' (if requested) to end, bounded by maxDrop
	lo := hit
	switch strings.ToLower(strings.TrimSpace(br.DropUntil)) {
	case "previous_blank_line":
		for lo > 0 && strings.TrimSpace(lines[lo-1]) != "" && (hit-lo) < maxDrop {
			lo--
		}
	default:
		// just cap by maxDrop
		if maxDrop > 0 {
			lo = maxInt(hit-maxDrop+1, 0)
		}
	}
	return lines[:lo]
}

func matchesBoundaryLine(s string, br *boundaryRule) bool {
	ts := strings.TrimSpace(s)
	if ts == "" {
		return false
	}
	for _, p := range br.StartsWith {
		if strings.HasPrefix(ts, p) {
			return true
		}
	}
	for _, p := range br.ContainsAny {
		if p != "" && strings.Contains(ts, p) {
			return true
		}
	}
	return false
}

// ---------- titling, anchors, dedupe ----------

func firstHeadingOrPathTitle(lines []string, rel string, generic *regexp.Regexp) string {
	for _, ln := range lines {
		if m := mdHeadingRE.FindStringSubmatch(ln); m != nil {
			title := strings.TrimSpace(stripMdLinks(m[1]))
			if generic != nil && generic.MatchString(title) {
				break
			}
			if isAllCaps(title) {
				break
			}
			return title
		}
	}
	base := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
	base = strings.ReplaceAll(strings.ReplaceAll(base, "-", " "), "_", " ")
	return strings.Title(base)
}

func dedupeByTitle(items []docItem) []docItem {
	seen := map[string]bool{}
	var out []docItem
	for _, it := range items {
		key := strings.ToLower(strings.TrimSpace(stripMdLinks(it.Title)))
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(it.RelPath))
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, it)
	}
	return out
}

func anchorFromTitle(t string) string {
	s := strings.ToLower(strings.TrimSpace(stripMdLinks(t)))
	s = strings.ReplaceAll(s, "&", "and")
	s = nonAnchorChar.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), "-")
}

// ---------- utils ----------

func parseRepoURL(u string) (string, string, string, string) {
	if u == "" {
		return "", "", "", ""
	}
	pu, err := url.Parse(u)
	if err != nil {
		return "", "", "", ""
	}
	parts := strings.Split(strings.Trim(pu.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", "", ""
	}
	owner := parts[0]
	repo := parts[1]
	var ref, p string
	if len(parts) >= 5 && parts[2] == "tree" {
		ref = parts[3]
		p = strings.Join(parts[4:], "/")
	}
	return owner, repo, ref, p
}

func parseCSVSet(csv string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, seg := range strings.Split(csv, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		if !strings.HasPrefix(seg, ".") {
			seg = "." + seg
		}
		out[strings.ToLower(seg)] = struct{}{}
	}
	return out
}

func sliceWithTrim(lines []string, trimHead, trimTail int) []string {
	start := 0
	end := len(lines)
	if len(lines) > trimHead {
		start = trimHead
	}
	if len(lines) > trimTail {
		end = len(lines) - trimTail
	}
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	return lines[start:end]
}

func hasNonEmpty(lines []string) bool {
	for _, ln := range lines {
		if strings.TrimSpace(ln) != "" {
			return true
		}
	}
	return false
}

func stripMdLinks(s string) string {
	// images: ![alt](src) -> alt
	s = mdImageRE.ReplaceAllStringFunc(s, func(m string) string {
		sub := mdImageRE.FindStringSubmatch(m)
		if len(sub) == 3 {
			return sub[1]
		}
		return m
	})
	// links: [text](href) -> text
	s = mdLinkRE.ReplaceAllStringFunc(s, func(m string) string {
		sub := mdLinkRE.FindStringSubmatch(m)
		if len(sub) == 3 {
			return sub[1]
		}
		return m
	})
	return s
}

func isAllCaps(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" {
		return false
	}
	hasLetter := false
	for _, r := range t {
		if r >= 'a' && r <= 'z' {
			return false
		}
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			hasLetter = true
		}
	}
	return hasLetter
}

func commonRunLenStart(items []docItem, quorum float64, maxProbe int) int {
	if len(items) == 0 {
		return 0
	}
	maxLen := 0
	for _, it := range items {
		if len(it.Lines) > maxLen {
			maxLen = len(it.Lines)
		}
	}
	if maxLen > maxProbe {
		maxLen = maxProbe
	}
	for L := maxLen; L > 0; L-- {
		counts := map[string]int{}
		for _, it := range items {
			if len(it.Lines) >= L {
				key := strings.Join(it.Lines[:L], "\n")
				counts[key]++
			}
		}
		most := 0
		var keyMost string
		for k, v := range counts {
			if v > most {
				most = v
				keyMost = k
			}
		}
		if most >= int(float64(len(items))*quorum+0.5) {
			nonempty := 0
			for _, ln := range strings.Split(keyMost, "\n") {
				if strings.TrimSpace(ln) != "" {
					nonempty++
				}
			}
			if nonempty >= max(2, L/3) {
				return L
			}
		}
	}
	return 0
}

func commonRunLenEnd(items []docItem, quorum float64, maxProbe int) int {
	if len(items) == 0 {
		return 0
	}
	maxLen := 0
	for _, it := range items {
		if len(it.Lines) > maxLen {
			maxLen = len(it.Lines)
		}
	}
	if maxLen > maxProbe {
		maxLen = maxProbe
	}
	for L := maxLen; L > 0; L-- {
		counts := map[string]int{}
		for _, it := range items {
			if len(it.Lines) >= L {
				key := strings.Join(it.Lines[len(it.Lines)-L:], "\n")
				counts[key]++
			}
		}
		most := 0
		var keyMost string
		for k, v := range counts {
			if v > most {
				most = v
				keyMost = k
			}
		}
		if most >= int(float64(len(items))*quorum+0.5) {
			nonempty := 0
			for _, ln := range strings.Split(keyMost, "\n") {
				if strings.TrimSpace(ln) != "" {
					nonempty++
				}
			}
			if nonempty >= max(2, L/3) {
				return L
			}
		}
	}
	return 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

// ---------- link rewriting ----------

func rewriteRelativeLinks(lines []string, owner, repo, ref, baseDir string) []string {
	var out []string
	for _, ln := range lines {
		// images first
		newLn := mdImageRE.ReplaceAllStringFunc(ln, func(m string) string {
			sub := mdImageRE.FindStringSubmatch(m)
			if len(sub) != 3 {
				return m
			}
			alt := sub[1]
			dest := strings.TrimSpace(sub[2])
			if shouldSkipRewrite(dest) {
				return m
			}
			res := path.Clean(path.Join("/", baseDir, dest))
			res = strings.TrimPrefix(res, "/")
			blob := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, ref, escapePathSegments(res))
			return fmt.Sprintf("![%s](%s)", alt, blob)
		})
		// then links
		newLn = mdLinkRE.ReplaceAllStringFunc(newLn, func(m string) string {
			sub := mdLinkRE.FindStringSubmatch(m)
			if len(sub) != 3 {
				return m
			}
			label := sub[1]
			dest := strings.TrimSpace(sub[2])
			if shouldSkipRewrite(dest) {
				return m
			}
			res := path.Clean(path.Join("/", baseDir, dest))
			res = strings.TrimPrefix(res, "/")
			blob := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, ref, escapePathSegments(res))
			return fmt.Sprintf("[%s](%s)", label, blob)
		})
		out = append(out, newLn)
	}
	return out
}

func shouldSkipRewrite(dest string) bool {
	return dest == "" ||
		strings.HasPrefix(dest, "#") ||
		strings.HasPrefix(dest, "mailto:") ||
		strings.HasPrefix(dest, "http://") ||
		strings.HasPrefix(dest, "https://") ||
		strings.HasPrefix(dest, "data:")
}

func escapePathSegments(p string) string {
	segs := strings.Split(p, "/")
	for i := range segs {
		segs[i] = url.PathEscape(segs[i])
	}
	return strings.Join(segs, "/")
}
