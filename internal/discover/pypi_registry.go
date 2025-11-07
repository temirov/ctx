package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type pypiPackageMetadata struct {
	ProjectURL string
	HomePage   string
}

type pypiRegistryClient interface {
	Metadata(ctx context.Context, name string) (pypiPackageMetadata, error)
}

type pypiHTTPRegistry struct {
	client httpClient
	base   string
	cache  map[string]pypiPackageMetadata
	mutex  sync.Mutex
}

func newPyPIRegistry(base string) pypiRegistryClient {
	return newPyPIRegistryWithClient(base, &http.Client{Timeout: 20 * time.Second})
}

func newPyPIRegistryWithClient(base string, client httpClient) pypiRegistryClient {
	if base == "" {
		base = "https://pypi.org/pypi"
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return &pypiHTTPRegistry{
		client: client,
		base:   strings.TrimRight(base, "/"),
		cache:  map[string]pypiPackageMetadata{},
	}
}

func (registry *pypiHTTPRegistry) Metadata(ctx context.Context, name string) (pypiPackageMetadata, error) {
	registry.mutex.Lock()
	if metadata, ok := registry.cache[name]; ok {
		registry.mutex.Unlock()
		return metadata, nil
	}
	registry.mutex.Unlock()

	requestURL := fmt.Sprintf("%s/%s/json", registry.base, url.PathEscape(name))
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if requestErr != nil {
		return pypiPackageMetadata{}, requestErr
	}
	response, responseErr := registry.client.Do(request)
	if responseErr != nil {
		return pypiPackageMetadata{}, responseErr
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return pypiPackageMetadata{}, fmt.Errorf("pypi registry returned %d for %s", response.StatusCode, name)
	}
	var payload pypiRegistryResponse
	if decodeErr := json.NewDecoder(response.Body).Decode(&payload); decodeErr != nil {
		return pypiPackageMetadata{}, fmt.Errorf("decode pypi payload for %s: %w", name, decodeErr)
	}
	projectURL := selectPyPIProjectURL(payload.Info.ProjectURLs)
	if projectURL == "" {
		projectURL = payload.Info.HomePage
	}
	metadata := pypiPackageMetadata{
		ProjectURL: projectURL,
		HomePage:   payload.Info.HomePage,
	}
	registry.mutex.Lock()
	registry.cache[name] = metadata
	registry.mutex.Unlock()
	return metadata, nil
}

type pypiRegistryResponse struct {
	Info pypiInfo `json:"info"`
}

type pypiInfo struct {
	HomePage    string            `json:"home_page"`
	ProjectURLs map[string]string `json:"project_urls"`
}

func selectPyPIProjectURL(urls map[string]string) string {
	priorities := []string{"Source", "Source Code", "Homepage", "Documentation"}
	for _, key := range priorities {
		if urls == nil {
			break
		}
		if value := strings.TrimSpace(urls[key]); value != "" {
			return value
		}
	}
	for _, value := range urls {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
