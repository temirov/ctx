package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestServerExecutesCommand(t *testing.T) {
	t.Parallel()

	responseBody := CommandResponse{
		Output:   "executed",
		Format:   "json",
		Warnings: []string{"notice"},
	}
	executors := map[string]CommandExecutor{
		"sample": CommandExecutorFunc(func(_ context.Context, request CommandRequest) (CommandResponse, error) {
			expectedPayload := `{"value":42}`
			if string(request.Payload) != expectedPayload {
				return CommandResponse{}, NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("unexpected payload: %s", string(request.Payload)))
			}
			return responseBody, nil
		}),
	}
	server := NewServer(Config{
		Address: "127.0.0.1:0",
		Capabilities: []Capability{
			{Name: "sample", Description: "Sample command"},
		},
		Executors:       executors,
		ShutdownTimeout: time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addressChan := make(chan string, 1)
	done := make(chan error, 1)

	go func() {
		done <- server.Run(ctx, func(address string) {
			addressChan <- address
		})
	}()

	address := waitForAddress(t, addressChan)

	client := http.Client{Timeout: 2 * time.Second}
	requestBody := bytes.NewBufferString(`{"value":42}`)
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+address+"/commands/sample", requestBody)
	if requestErr != nil {
		t.Fatalf("create request: %v", requestErr)
	}

	response, responseErr := client.Do(request)
	if responseErr != nil {
		t.Fatalf("execute request: %v", responseErr)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", response.StatusCode)
	}

	var decoded CommandResponse
	if decodeErr := json.NewDecoder(response.Body).Decode(&decoded); decodeErr != nil {
		t.Fatalf("decode response: %v", decodeErr)
	}

	if decoded.Output != responseBody.Output {
		t.Fatalf("unexpected output: %s", decoded.Output)
	}
	if decoded.Format != responseBody.Format {
		t.Fatalf("unexpected format: %s", decoded.Format)
	}
	if len(decoded.Warnings) != len(responseBody.Warnings) || decoded.Warnings[0] != responseBody.Warnings[0] {
		t.Fatalf("unexpected warnings: %+v", decoded.Warnings)
	}

	cancel()
	if runErr := <-done; runErr != nil {
		t.Fatalf("server shutdown: %v", runErr)
	}
}

func TestServerHandlesUnknownCommand(t *testing.T) {
	t.Parallel()

	server := NewServer(Config{
		Address: "127.0.0.1:0",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addressChan := make(chan string, 1)
	done := make(chan error, 1)

	go func() {
		done <- server.Run(ctx, func(address string) {
			addressChan <- address
		})
	}()

	address := waitForAddress(t, addressChan)

	client := http.Client{Timeout: 2 * time.Second}
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+address+"/commands/missing", http.NoBody)
	if requestErr != nil {
		t.Fatalf("create request: %v", requestErr)
	}

	response, responseErr := client.Do(request)
	if responseErr != nil {
		t.Fatalf("execute request: %v", responseErr)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected status code: %d", response.StatusCode)
	}

	cancel()
	if runErr := <-done; runErr != nil {
		t.Fatalf("server shutdown: %v", runErr)
	}
}

func TestServerPropagatesExecutorStatus(t *testing.T) {
	t.Parallel()

	executors := map[string]CommandExecutor{
		"fail": CommandExecutorFunc(func(_ context.Context, _ CommandRequest) (CommandResponse, error) {
			return CommandResponse{}, NewCommandExecutionError(http.StatusTeapot, fmt.Errorf("custom failure"))
		}),
	}
	server := NewServer(Config{
		Address:   "127.0.0.1:0",
		Executors: executors,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addressChan := make(chan string, 1)
	done := make(chan error, 1)

	go func() {
		done <- server.Run(ctx, func(address string) {
			addressChan <- address
		})
	}()

	address := waitForAddress(t, addressChan)

	client := http.Client{Timeout: 2 * time.Second}
	request, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+address+"/commands/fail", http.NoBody)
	if requestErr != nil {
		t.Fatalf("create request: %v", requestErr)
	}

	response, responseErr := client.Do(request)
	if responseErr != nil {
		t.Fatalf("execute request: %v", responseErr)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusTeapot {
		t.Fatalf("unexpected status code: %d", response.StatusCode)
	}

	cancel()
	if runErr := <-done; runErr != nil {
		t.Fatalf("server shutdown: %v", runErr)
	}
}

func waitForAddress(t *testing.T, addressChan <-chan string) string {
	t.Helper()
	select {
	case address := <-addressChan:
		return address
	case <-time.After(3 * time.Second):
		t.Fatalf("server address not received")
		return ""
	}
}
