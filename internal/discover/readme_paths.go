package discover

import (
	"path"
	"regexp"
	"strings"
)

var markdownLinkPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)\s]+)\)`)

func extractDocPathsFromReadme(content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	matches := markdownLinkPattern.FindAllStringSubmatch(trimmed, -1)
	if len(matches) == 0 {
		return nil
	}
	const maxHints = 5
	seen := map[string]struct{}{}
	var results []string
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		candidate := normalizeReadmePath(match[1])
		if candidate == "" {
			continue
		}
		lowerCandidate := strings.ToLower(candidate)
		if _, ok := seen[lowerCandidate]; ok {
			continue
		}
		addCandidate := func(value string) {
			if value == "" {
				return
			}
			key := strings.ToLower(value)
			if _, exists := seen[key]; exists {
				return
			}
			seen[key] = struct{}{}
			results = append(results, value)
		}
		if strings.HasSuffix(lowerCandidate, ".md") || strings.HasSuffix(lowerCandidate, ".markdown") {
			if dir := path.Dir(candidate); dir != "." {
				addCandidate(dir)
				if len(results) >= maxHints {
					break
				}
			}
		}
		addCandidate(candidate)
		if len(results) >= maxHints {
			break
		}
	}
	return results
}

func normalizeReadmePath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	lowered := strings.ToLower(trimmed)
	if strings.HasPrefix(lowered, "http://") || strings.HasPrefix(lowered, "https://") || strings.HasPrefix(lowered, "mailto:") {
		return ""
	}
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "?") {
		return ""
	}
	cleaned := strings.Split(trimmed, "#")[0]
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return ""
	}
	if strings.Contains(cleaned, "://") {
		return ""
	}
	cleaned = strings.TrimPrefix(cleaned, "./")
	cleaned = strings.TrimPrefix(cleaned, "/")
	if strings.Contains(cleaned, "..") {
		return ""
	}
	cleaned = path.Clean(cleaned)
	if cleaned == "." {
		return ""
	}
	lowercase := strings.ToLower(cleaned)
	if strings.HasSuffix(lowercase, ".md") || strings.HasSuffix(lowercase, ".markdown") {
		return cleaned
	}
	keywords := []string{"doc", "guide", "manual", "handbook", "site", "tutorial"}
	for _, keyword := range keywords {
		if strings.Contains(lowercase, keyword) {
			return cleaned
		}
	}
	return ""
}
