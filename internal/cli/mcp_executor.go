package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/temirov/ctx/internal/services/mcp"
	"github.com/temirov/ctx/internal/types"
)

type streamRequestCommon struct {
	Paths          []string             `json:"paths"`
	Exclude        []string             `json:"exclude"`
	UseGitignore   *bool                `json:"useGitignore"`
	UseIgnoreFile  *bool                `json:"useIgnoreFile"`
	IncludeGit     *bool                `json:"includeGit"`
	Format         string               `json:"format"`
	Summary        *bool                `json:"summary"`
	IncludeContent *bool                `json:"includeContent"`
	Documentation  *bool                `json:"documentation"`
	Tokens         *streamTokensRequest `json:"tokens"`
}

type streamTokensRequest struct {
	Enabled *bool  `json:"enabled"`
	Model   string `json:"model"`
}

type streamConfigurationDefaults struct {
	includeContent      bool
	summary             bool
	documentation       bool
	allowDocumentation  bool
	commandName         string
	allowContentControl bool
}

type streamExecutionParameters struct {
	commandName          string
	paths                []string
	exclusionPatterns    []string
	useGitignore         bool
	useIgnoreFile        bool
	includeGit           bool
	format               string
	documentationEnabled bool
	summaryEnabled       bool
	includeContent       bool
	tokenConfiguration   tokenOptions
}

type callChainRequest struct {
	Target        string `json:"target"`
	Depth         *int   `json:"depth"`
	Format        string `json:"format"`
	Documentation *bool  `json:"documentation"`
}

func mcpCommandExecutors() map[string]mcp.CommandExecutor {
	return map[string]mcp.CommandExecutor{
		types.CommandTree:      mcp.CommandExecutorFunc(executeTreeCommand),
		types.CommandContent:   mcp.CommandExecutorFunc(executeContentCommand),
		types.CommandCallChain: mcp.CommandExecutorFunc(executeCallChainCommand),
	}
}

func executeTreeCommand(commandContext context.Context, request mcp.CommandRequest) (mcp.CommandResponse, error) {
	parameters, parseErr := parseStreamRequest(
		request.Payload,
		streamConfigurationDefaults{
			includeContent:      false,
			summary:             true,
			documentation:       false,
			allowDocumentation:  false,
			commandName:         types.CommandTree,
			allowContentControl: true,
		},
	)
	if parseErr != nil {
		return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("decode tree request: %w", parseErr))
	}
	return invokeStreamCommand(commandContext, parameters)
}

func executeContentCommand(commandContext context.Context, request mcp.CommandRequest) (mcp.CommandResponse, error) {
	parameters, parseErr := parseStreamRequest(
		request.Payload,
		streamConfigurationDefaults{
			includeContent:      true,
			summary:             true,
			documentation:       false,
			allowDocumentation:  true,
			commandName:         types.CommandContent,
			allowContentControl: true,
		},
	)
	if parseErr != nil {
		return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("decode content request: %w", parseErr))
	}
	return invokeStreamCommand(commandContext, parameters)
}

func executeCallChainCommand(commandContext context.Context, request mcp.CommandRequest) (mcp.CommandResponse, error) {
	var payload callChainRequest
	if len(request.Payload) > 0 {
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("decode callchain request: %w", err))
		}
	}
	target := strings.TrimSpace(payload.Target)
	if target == "" {
		return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("target is required"))
	}
	format := payload.Format
	if format == "" {
		format = types.FormatJSON
	}
	format = strings.ToLower(format)
	if !isSupportedFormat(format) {
		return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf(invalidFormatMessage, format))
	}
	depth := defaultCallChainDepth
	if payload.Depth != nil {
		if *payload.Depth < 1 {
			return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("depth must be positive"))
		}
		depth = *payload.Depth
	}
	documentationEnabled := false
	if payload.Documentation != nil {
		documentationEnabled = *payload.Documentation
	}
	var outputBuffer bytes.Buffer
	var warningBuffer bytes.Buffer
	executionErr := runTool(
		commandContext,
		types.CommandCallChain,
		[]string{target},
		nil,
		true,
		true,
		false,
		depth,
		format,
		documentationEnabled,
		false,
		false,
		tokenOptions{},
		&outputBuffer,
		&warningBuffer,
		false,
		nil,
	)
	if executionErr != nil {
		return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("execute callchain: %w", executionErr))
	}
	return mcp.CommandResponse{
		Output:   outputBuffer.String(),
		Format:   format,
		Warnings: extractWarnings(&warningBuffer),
	}, nil
}

func parseStreamRequest(payload json.RawMessage, defaults streamConfigurationDefaults) (streamExecutionParameters, error) {
	var requestBody streamRequestCommon
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &requestBody); err != nil {
			return streamExecutionParameters{}, err
		}
	}
	if !defaults.allowDocumentation && requestBody.Documentation != nil {
		return streamExecutionParameters{}, fmt.Errorf("documentation is not supported for %s", defaults.commandName)
	}
	paths := sanitizePaths(requestBody.Paths)
	format := requestBody.Format
	if format == "" {
		format = types.FormatJSON
	}
	format = strings.ToLower(format)
	if !isSupportedFormat(format) {
		return streamExecutionParameters{}, fmt.Errorf(invalidFormatMessage, format)
	}
	useGitignore := resolveBoolean(requestBody.UseGitignore, true)
	useIgnoreFile := resolveBoolean(requestBody.UseIgnoreFile, true)
	includeGit := resolveBoolean(requestBody.IncludeGit, false)
	summaryEnabled := resolveBoolean(requestBody.Summary, defaults.summary)
	includeContent := defaults.includeContent
	if defaults.allowContentControl && requestBody.IncludeContent != nil {
		includeContent = *requestBody.IncludeContent
	}
	documentationEnabled := defaults.documentation
	if defaults.allowDocumentation && requestBody.Documentation != nil {
		documentationEnabled = *requestBody.Documentation
	}

	tokenConfiguration := tokenOptions{
		enabled: false,
		model:   defaultTokenizerModelName,
	}
	if requestBody.Tokens != nil {
		if requestBody.Tokens.Enabled != nil {
			tokenConfiguration.enabled = *requestBody.Tokens.Enabled
		}
		if requestBody.Tokens.Model != "" {
			tokenConfiguration.model = requestBody.Tokens.Model
		}
	}

	return streamExecutionParameters{
		commandName:          defaults.commandName,
		paths:                paths,
		exclusionPatterns:    append([]string{}, requestBody.Exclude...),
		useGitignore:         useGitignore,
		useIgnoreFile:        useIgnoreFile,
		includeGit:           includeGit,
		format:               format,
		documentationEnabled: documentationEnabled,
		summaryEnabled:       summaryEnabled,
		includeContent:       includeContent,
		tokenConfiguration:   tokenConfiguration,
	}, nil
}

func invokeStreamCommand(commandContext context.Context, parameters streamExecutionParameters) (mcp.CommandResponse, error) {
	var outputBuffer bytes.Buffer
	var warningBuffer bytes.Buffer
	executionErr := runTool(
		commandContext,
		parameters.commandName,
		parameters.paths,
		parameters.exclusionPatterns,
		parameters.useGitignore,
		parameters.useIgnoreFile,
		parameters.includeGit,
		defaultCallChainDepth,
		parameters.format,
		parameters.documentationEnabled,
		parameters.summaryEnabled,
		parameters.includeContent,
		parameters.tokenConfiguration,
		&outputBuffer,
		&warningBuffer,
		false,
		nil,
	)
	if executionErr != nil {
		return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("execute %s: %w", parameters.commandName, executionErr))
	}
	return mcp.CommandResponse{
		Output:   outputBuffer.String(),
		Format:   parameters.format,
		Warnings: extractWarnings(&warningBuffer),
	}, nil
}

func sanitizePaths(input []string) []string {
	if len(input) == 0 {
		return []string{defaultPath}
	}
	var result []string
	for _, candidate := range input {
		trimmed := strings.TrimSpace(candidate)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return []string{defaultPath}
	}
	return result
}

func resolveBoolean(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

func extractWarnings(buffer *bytes.Buffer) []string {
	if buffer == nil {
		return nil
	}
	trimmed := strings.TrimSpace(buffer.String())
	if trimmed == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	var warnings []string
	for _, line := range lines {
		clean := strings.TrimSpace(line)
		if clean != "" {
			warnings = append(warnings, clean)
		}
	}
	if len(warnings) == 0 {
		return nil
	}
	return warnings
}
