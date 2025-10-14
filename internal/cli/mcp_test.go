package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/temirov/ctx/internal/services/mcp"
)

func TestStartMCPServerServesCapabilities(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
	}{{name: "serves capabilities"}}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var buffer bytes.Buffer
			done := make(chan error, 1)

			go func() {
				done <- startMCPServer(ctx, &buffer)
			}()

			address := waitForMCPAddress(t, &buffer)

			request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address+"/capabilities", nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}

			client := http.Client{Timeout: 2 * time.Second}
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
				t.Fatalf("decode body: %v", err)
			}

			expected := mcpCapabilities()
			if len(body.Capabilities) != len(expected) {
				t.Fatalf("expected %d capabilities, got %d", len(expected), len(body.Capabilities))
			}

			for index, capability := range expected {
				payload := body.Capabilities[index]
				if payload != capability {
					t.Fatalf("capability %d mismatch: got %+v, want %+v", index, payload, capability)
				}
			}

			cancel()
			if err := <-done; err != nil {
				t.Fatalf("server shutdown error: %v", err)
			}
		})
	}
}

func waitForMCPAddress(t *testing.T, buffer *bytes.Buffer) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		output := buffer.String()
		if output != "" {
			for _, line := range strings.Split(output, "\n") {
				if strings.HasPrefix(line, "MCP server listening on ") {
					return strings.TrimPrefix(line, "MCP server listening on ")
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server address not reported: %s", buffer.String())
	return ""
}
