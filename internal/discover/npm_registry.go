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

type httpClient interface {
	Do(request *http.Request) (*http.Response, error)
}

type npmPackageMetadata struct {
	RepositoryURL string
	Homepage      string
}

type npmRegistryClient interface {
	Metadata(ctx context.Context, name string) (npmPackageMetadata, error)
}

type npmHTTPRegistry struct {
	client httpClient
	base   string
	cache  map[string]npmPackageMetadata
	mutex  sync.Mutex
}

func newNPMRegistry(base string) npmHTTPRegistry {
	return newNPMRegistryWithClient(base, &http.Client{Timeout: 20 * time.Second})
}

func newNPMRegistryWithClient(base string, client httpClient) npmHTTPRegistry {
	if base == "" {
		base = "https://registry.npmjs.org"
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return npmHTTPRegistry{
		client: client,
		base:   strings.TrimRight(base, "/"),
		cache:  map[string]npmPackageMetadata{},
	}
}

func (registry npmHTTPRegistry) Metadata(ctx context.Context, name string) (npmPackageMetadata, error) {
	registry.mutex.Lock()
	if metadata, ok := registry.cache[name]; ok {
		registry.mutex.Unlock()
		return metadata, nil
	}
	registry.mutex.Unlock()

	requestURL := fmt.Sprintf("%s/%s", registry.base, url.PathEscape(name))
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if requestErr != nil {
		return npmPackageMetadata{}, requestErr
	}
	response, responseErr := registry.client.Do(request)
	if responseErr != nil {
		return npmPackageMetadata{}, responseErr
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return npmPackageMetadata{}, fmt.Errorf("npm registry returned %d for %s", response.StatusCode, name)
	}
	var payload npmRegistryResponse
	if decodeErr := json.NewDecoder(response.Body).Decode(&payload); decodeErr != nil {
		return npmPackageMetadata{}, fmt.Errorf("decode npm payload for %s: %w", name, decodeErr)
	}
	metadata := npmPackageMetadata{
		RepositoryURL: payload.Repository.URL,
		Homepage:      payload.Homepage,
	}
	if metadata.RepositoryURL == "" && payload.LatestVersion.Repository.URL != "" {
		metadata.RepositoryURL = payload.LatestVersion.Repository.URL
	}
	if metadata.Homepage == "" && payload.LatestVersion.Homepage != "" {
		metadata.Homepage = payload.LatestVersion.Homepage
	}
	registry.mutex.Lock()
	registry.cache[name] = metadata
	registry.mutex.Unlock()
	return metadata, nil
}

type npmRepository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type npmVersion struct {
	Repository npmRepository `json:"repository"`
	Homepage   string        `json:"homepage"`
}

type npmRegistryResponse struct {
	Repository    npmRepository         `json:"repository"`
	Homepage      string                `json:"homepage"`
	DistTags      map[string]string     `json:"dist-tags"`
	Versions      map[string]npmVersion `json:"versions"`
	LatestVersion npmVersion
}

func (response *npmRegistryResponse) UnmarshalJSON(data []byte) error {
	type alias npmRegistryResponse
	aux := struct {
		*alias
	}{
		alias: (*alias)(response),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	latestTag := ""
	if aux.DistTags != nil {
		latestTag = aux.DistTags["latest"]
	}
	if latestTag != "" && aux.Versions != nil {
		if version, ok := aux.Versions[latestTag]; ok {
			response.LatestVersion = version
			return nil
		}
	}
	for _, version := range aux.Versions {
		response.LatestVersion = version
		break
	}
	return nil
}
