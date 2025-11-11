package webdoc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetcherRespectsDepthAndSameHost(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/root", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintf(writer, `<html><head><title>Root</title></head><body><p>Welcome</p><a href="/child">Child</a><a href="https://example.org/external">External</a></body></html>`)
	})
	mux.HandleFunc("/child", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintf(writer, `<html><head><title>Child</title></head><body><p>Child body</p></body></html>`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	fetcher := NewFetcher(server.Client())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pages, err := fetcher.Fetch(ctx, server.URL+"/root", 1)
	if err != nil {
		t.Fatalf("expected fetch to succeed, got %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
	if pages[0].URL != server.URL+"/root" {
		t.Fatalf("expected first page to be root, got %s", pages[0].URL)
	}
	if pages[1].URL != server.URL+"/child" {
		t.Fatalf("expected second page to be child link, got %s", pages[1].URL)
	}

	rootOnly, err := fetcher.Fetch(ctx, server.URL+"/root", 0)
	if err != nil {
		t.Fatalf("expected depth 0 fetch to succeed, got %v", err)
	}
	if len(rootOnly) != 1 {
		t.Fatalf("expected only root page at depth 0, got %d", len(rootOnly))
	}
}

func TestFetcherSanitizesMarkup(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintf(writer, `
		<html>
		<head><title>Sample</title></head>
		<body>
			<script>console.log('skip me')</script>
			<style>body { color: red; }</style>
			<h1>Heading</h1>
			<p>Paragraph with <code>fmt.Println("hi")</code></p>
			<ul><li>Item one</li></ul>
		</body>
		</html>`)
	}))
	defer server.Close()

	fetcher := NewFetcher(server.Client())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pages, err := fetcher.Fetch(ctx, server.URL, 0)
	if err != nil {
		t.Fatalf("expected fetch to succeed, got %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected single page, got %d", len(pages))
	}
	contents := pages[0].Content
	if strings.Contains(contents, "console.log") || strings.Contains(contents, "<script") {
		t.Fatalf("expected scripts to be stripped, got %q", contents)
	}
	if !strings.Contains(contents, "Heading") {
		t.Fatalf("expected heading to remain, got %q", contents)
	}
	if !strings.Contains(contents, "fmt.Println") {
		t.Fatalf("expected code text to remain, got %q", contents)
	}
	if !strings.Contains(contents, "Item one") {
		t.Fatalf("expected list items to remain, got %q", contents)
	}
}
