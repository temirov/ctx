package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/temirov/ctx/internal/services/mcp"
)

func TestServerRunExposesCapabilities(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		config       mcp.Config
		expectedCaps []mcp.Capability
	}{
		{
			name: "single capability",
			config: mcp.Config{
				Capabilities: []mcp.Capability{
					{Name: "tree", Description: "List directories"},
				},
				Address: "127.0.0.1:0",
			},
			expectedCaps: []mcp.Capability{{Name: "tree", Description: "List directories"}},
		},
		{
			name: "multiple capabilities",
			config: mcp.Config{
				Capabilities: []mcp.Capability{
					{Name: "content", Description: "Show file content"},
					{Name: "callchain", Description: "Inspect call relationships"},
				},
			},
			expectedCaps: []mcp.Capability{
				{Name: "content", Description: "Show file content"},
				{Name: "callchain", Description: "Inspect call relationships"},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			server := mcp.NewServer(testCase.config)
			addressCh := make(chan string, 1)
			errorCh := make(chan error, 1)

			go func() {
				errorCh <- server.Run(ctx, func(address string) {
					addressCh <- address
				})
			}()

			select {
			case address := <-addressCh:
				client := http.Client{Timeout: 2 * time.Second}
				request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address+"/capabilities", nil)
				if err != nil {
					t.Fatalf("new request: %v", err)
				}
				response, err := client.Do(request)
				if err != nil {
					t.Fatalf("perform request: %v", err)
				}
				defer response.Body.Close()

				if response.StatusCode != http.StatusOK {
					t.Fatalf("unexpected status: %d", response.StatusCode)
				}

				var body struct {
					Capabilities []mcp.Capability `json:"capabilities"`
				}
				if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
					t.Fatalf("decode response: %v", err)
				}

				if len(body.Capabilities) != len(testCase.expectedCaps) {
					t.Fatalf("expected %d capabilities, got %d", len(testCase.expectedCaps), len(body.Capabilities))
				}
				for index, capability := range body.Capabilities {
					expected := testCase.expectedCaps[index]
					if capability != expected {
						t.Fatalf("capability %d mismatch: got %+v, want %+v", index, capability, expected)
					}
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("server did not start")
			}

			cancel()
			if err := <-errorCh; err != nil {
				t.Fatalf("server error: %v", err)
			}
		})
	}
}
