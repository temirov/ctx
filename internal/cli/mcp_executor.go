package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/temirov/ctx/internal/docs/githubdoc"
	"github.com/temirov/ctx/internal/services/mcp"
	"github.com/temirov/ctx/internal/types"
)

type streamRequestCommon struct {
	Paths          []string             `json:"paths"`
	Exclude        []string             `json:"exclude"`
	UseGitignore   *bool                `json:"useGitignore"`
	UseIgnoreFile  *bool                `json:"useIgnoreFile"`
	IncludeGit     *bool                `json:"includeGit"`
	Summary        *bool                `json:"summary"`
	IncludeContent *bool                `json:"includeContent"`
	Documentation  json.RawMessage      `json:"documentation"`
	Tokens         *streamTokensRequest `json:"tokens"`
}

type streamTokensRequest struct {
	Enabled *bool  `json:"enabled"`
	Model   string `json:"model"`
}

type streamConfigurationDefaults struct {
	includeContent      bool
	summary             bool
	documentationMode   string
	allowDocumentation  bool
	commandName         string
	allowContentControl bool
}

type streamExecutionParameters struct {
	commandName        string
	paths              []string
	exclusionPatterns  []string
	useGitignore       bool
	useIgnoreFile      bool
	includeGit         bool
	format             string
	documentationMode  string
	summaryEnabled     bool
	includeContent     bool
	tokenConfiguration tokenOptions
}

type callChainRequest struct {
	Target        string          `json:"target"`
	Depth         *int            `json:"depth"`
	Documentation json.RawMessage `json:"documentation"`
}

type docRequest struct {
	RepositoryURL string          `json:"repoUrl"`
	Owner         string          `json:"owner"`
	Repository    string          `json:"repo"`
	Reference     string          `json:"ref"`
	Path          string          `json:"path"`
	Rules         string          `json:"rules"`
	Documentation json.RawMessage `json:"documentation"`
	APIBase       string          `json:"apiBase"`
}

func mcpCommandExecutors() map[string]mcp.CommandExecutor {
	return map[string]mcp.CommandExecutor{
		types.CommandTree:      mcp.CommandExecutorFunc(executeTreeCommand),
		types.CommandContent:   mcp.CommandExecutorFunc(executeContentCommand),
		types.CommandCallChain: mcp.CommandExecutorFunc(executeCallChainCommand),
		types.CommandDoc:       mcp.CommandExecutorFunc(executeDocCommand),
	}
}

func executeTreeCommand(commandContext context.Context, request mcp.CommandRequest) (mcp.CommandResponse, error) {
	parameters, parseErr := parseStreamRequest(
		request.Payload,
		streamConfigurationDefaults{
			includeContent:      false,
			summary:             true,
			documentationMode:   types.DocumentationModeDisabled,
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
			documentationMode:   types.DocumentationModeDisabled,
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
	format := types.FormatJSON
	depth := defaultCallChainDepth
	if payload.Depth != nil {
		if *payload.Depth < 1 {
			return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("depth must be positive"))
		}
		depth = *payload.Depth
	}
	documentationMode := types.DocumentationModeDisabled
	if len(payload.Documentation) > 0 {
		mode, modeErr := decodeDocumentationMode(payload.Documentation, documentationMode)
		if modeErr != nil {
			return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("decode documentation mode: %w", modeErr))
		}
		documentationMode = mode
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
		documentationMode,
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

func executeDocCommand(commandContext context.Context, request mcp.CommandRequest) (mcp.CommandResponse, error) {
	var payload docRequest
	if len(request.Payload) > 0 {
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("decode doc request: %w", err))
		}
	}
	mode, modeErr := normalizeDocumentationMode(types.DocumentationModeFull)
	if modeErr != nil {
		return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("resolve documentation mode: %w", modeErr))
	}
	if len(payload.Documentation) > 0 {
		resolvedMode, decodeErr := decodeDocumentationMode(payload.Documentation, mode)
		if decodeErr != nil {
			return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("decode documentation mode: %w", decodeErr))
		}
		mode = resolvedMode
	}
	coordinates, coordinatesErr := resolveRepositoryCoordinates(payload.RepositoryURL, payload.Owner, payload.Repository, payload.Reference, payload.Path)
	if coordinatesErr != nil {
		return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("resolve repository: %w", coordinatesErr))
	}
	var ruleSet githubdoc.RuleSet
	if payload.Rules != "" {
		loadedRuleSet, loadErr := githubdoc.LoadRuleSet(payload.Rules)
		if loadErr != nil {
			return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("load rules: %w", loadErr))
		}
		ruleSet = loadedRuleSet
	}
	var outputBuffer bytes.Buffer
	options := docCommandOptions{
		Coordinates:      coordinates,
		RuleSet:          ruleSet,
		Mode:             mode,
		APIBase:          payload.APIBase,
		ClipboardEnabled: false,
		Clipboard:        nil,
		Writer:           &outputBuffer,
	}
	if runErr := runDocCommand(commandContext, options); runErr != nil {
		return mcp.CommandResponse{}, mcp.NewCommandExecutionError(http.StatusBadRequest, fmt.Errorf("execute doc: %w", runErr))
	}
	return mcp.CommandResponse{
		Output: outputBuffer.String(),
		Format: types.FormatRaw,
	}, nil
}

func parseStreamRequest(payload json.RawMessage, defaults streamConfigurationDefaults) (streamExecutionParameters, error) {
	var requestBody streamRequestCommon
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &requestBody); err != nil {
			return streamExecutionParameters{}, err
		}
	}
	paths := sanitizePaths(requestBody.Paths)
	useGitignore := resolveBoolean(requestBody.UseGitignore, true)
	useIgnoreFile := resolveBoolean(requestBody.UseIgnoreFile, true)
	includeGit := resolveBoolean(requestBody.IncludeGit, false)
	summaryEnabled := resolveBoolean(requestBody.Summary, defaults.summary)
	includeContent := defaults.includeContent
	if defaults.allowContentControl && requestBody.IncludeContent != nil {
		includeContent = *requestBody.IncludeContent
	}
	documentationMode := defaults.documentationMode
	var documentationErr error
	if !defaults.allowDocumentation && len(requestBody.Documentation) > 0 {
		return streamExecutionParameters{}, fmt.Errorf("documentation is not supported for %s", defaults.commandName)
	}
	documentationMode, documentationErr = normalizeDocumentationMode(documentationMode)
	if documentationErr != nil {
		return streamExecutionParameters{}, documentationErr
	}
	if defaults.allowDocumentation && len(requestBody.Documentation) > 0 {
		documentationMode, documentationErr = decodeDocumentationMode(requestBody.Documentation, documentationMode)
		if documentationErr != nil {
			return streamExecutionParameters{}, documentationErr
		}
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
		commandName:        defaults.commandName,
		paths:              paths,
		exclusionPatterns:  append([]string{}, requestBody.Exclude...),
		useGitignore:       useGitignore,
		useIgnoreFile:      useIgnoreFile,
		includeGit:         includeGit,
		format:             types.FormatJSON,
		documentationMode:  documentationMode,
		summaryEnabled:     summaryEnabled,
		includeContent:     includeContent,
		tokenConfiguration: tokenConfiguration,
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
		parameters.documentationMode,
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

func decodeDocumentationMode(raw json.RawMessage, fallback string) (string, error) {
	mode, err := normalizeDocumentationMode(fallback)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 {
		return mode, nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		if asString == "" {
			return mode, nil
		}
		return normalizeDocumentationMode(asString)
	}
	var asBool bool
	if err := json.Unmarshal(raw, &asBool); err == nil {
		if asBool {
			return types.DocumentationModeRelevant, nil
		}
		return types.DocumentationModeDisabled, nil
	}
	return "", fmt.Errorf(invalidDocumentationModeMessage, string(raw))
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
