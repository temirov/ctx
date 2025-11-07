package discover

import "testing"

func TestExtractDocPathsFromReadme(t *testing.T) {
	content := `See the [full docs](docs/README.md) or the [guide](website/content/guide.md).
External [site](https://example.com/docs) should be ignored.`
	paths := extractDocPathsFromReadme(content)
	expected := []string{"docs", "docs/README.md", "website/content", "website/content/guide.md"}
	if len(paths) != len(expected) {
		t.Fatalf("expected %d paths, got %v", len(expected), paths)
	}
	for index, path := range expected {
		if paths[index] != path {
			t.Fatalf("expected %s at %d, got %s", path, index, paths[index])
		}
	}
}

func TestNormalizeReadmePathFiltersInvalid(t *testing.T) {
	cases := map[string]string{
		"http://example.com/docs": "",
		"#section":                "",
		"./docs/guide.md":         "docs/guide.md",
		"site/content":            "site/content",
		"../secret":               "",
	}
	for input, expected := range cases {
		if normalizeReadmePath(input) != expected {
			t.Fatalf("expected %q for %q, got %q", expected, input, normalizeReadmePath(input))
		}
	}
}
