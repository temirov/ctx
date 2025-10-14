package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	defaultListenAddress    = "127.0.0.1:0"
	defaultShutdownDuration = 5 * time.Second
)

// Capability describes a feature exposed by the MCP server.
type Capability struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Config defines runtime options for the MCP server.
type Config struct {
	Address         string
	Capabilities    []Capability
	ShutdownTimeout time.Duration
}

// Server serves capability metadata over HTTP.
type Server struct {
	config Config
}

// NewServer creates a new Server with defaults applied.
func NewServer(config Config) Server {
	normalized := config
	if normalized.Address == "" {
		normalized.Address = defaultListenAddress
	}
	if normalized.ShutdownTimeout <= 0 {
		normalized.ShutdownTimeout = defaultShutdownDuration
	}
	if normalized.Capabilities == nil {
		normalized.Capabilities = []Capability{}
	}
	return Server{config: normalized}
}

// Run starts the MCP server and blocks until the provided context is canceled.
// The notify callback receives the bound address once the listener is active.
func (server Server) Run(ctx context.Context, notify func(string)) error {
	listener, listenErr := net.Listen("tcp", server.config.Address)
	if listenErr != nil {
		return fmt.Errorf("listen on %s: %w", server.config.Address, listenErr)
	}
	actualAddress := listener.Addr().String()

	router := http.NewServeMux()
	router.HandleFunc("/capabilities", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		response := struct {
			Capabilities []Capability `json:"capabilities"`
		}{Capabilities: server.config.Capabilities}
		if encodeErr := json.NewEncoder(writer).Encode(response); encodeErr != nil {
			http.Error(writer, fmt.Sprintf("encode capabilities: %v", encodeErr), http.StatusInternalServerError)
		}
	})
	router.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})

	httpServer := &http.Server{Handler: router}
	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		serveErr := httpServer.Serve(listener)
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return fmt.Errorf("serve MCP: %w", serveErr)
		}
		return nil
	})

	if notify != nil {
		notify(actualAddress)
	}

	group.Go(func() error {
		<-groupCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), server.config.ShutdownTimeout)
		defer cancel()
		shutdownErr := httpServer.Shutdown(shutdownCtx)
		if shutdownErr != nil && !errors.Is(shutdownErr, context.Canceled) && !errors.Is(shutdownErr, http.ErrServerClosed) {
			return fmt.Errorf("shutdown MCP: %w", shutdownErr)
		}
		return nil
	})

	return group.Wait()
}
