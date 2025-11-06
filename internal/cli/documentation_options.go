package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/tyemirov/ctx/internal/docs"
	"github.com/tyemirov/ctx/internal/types"
)

var errGitHubTokenMissing = errors.New("github authorization token not found")

type documentationOptionsParameters struct {
	Mode          string
	RemoteEnabled bool
	RemoteAPIBase string
	TokenResolver githubTokenResolver
}

type documentationOptions struct {
	mode               string
	authorizationToken string
	remote             documentationRemoteOptions
}

type documentationRemoteOptions struct {
	enabled            bool
	apiBase            string
	authorizationToken string
}

func newDocumentationOptions(params documentationOptionsParameters) (documentationOptions, error) {
	normalizedMode, modeErr := normalizeDocumentationMode(params.Mode)
	if modeErr != nil {
		return documentationOptions{}, fmt.Errorf("normalize documentation mode: %w", modeErr)
	}

	var authorizationToken string
	if params.TokenResolver != nil {
		token, tokenErr := params.TokenResolver.Resolve()
		if tokenErr != nil {
			if params.RemoteEnabled || !errors.Is(tokenErr, errGitHubTokenMissing) {
				return documentationOptions{}, fmt.Errorf("resolve GitHub token: %w", tokenErr)
			}
		} else {
			authorizationToken = token
		}
	}

	result := documentationOptions{
		mode:               normalizedMode,
		authorizationToken: authorizationToken,
	}

	if params.RemoteEnabled {
		if normalizedMode != types.DocumentationModeFull {
			return documentationOptions{}, fmt.Errorf("remote documentation requires %s mode", types.DocumentationModeFull)
		}
		if authorizationToken == "" {
			return documentationOptions{}, fmt.Errorf("resolve GitHub token: %w", errGitHubTokenMissing)
		}
		result.remote = documentationRemoteOptions{
			enabled:            true,
			apiBase:            strings.TrimSpace(params.RemoteAPIBase),
			authorizationToken: authorizationToken,
		}
	}

	return result, nil
}

func (options documentationOptions) Mode() string {
	if options.mode == "" {
		return types.DocumentationModeDisabled
	}
	return options.mode
}

func (options documentationOptions) AuthorizationToken() string {
	return options.authorizationToken
}

func (options documentationOptions) CollectorOptions() docs.CollectorOptions {
	if !options.remote.enabled {
		return docs.CollectorOptions{}
	}
	return docs.CollectorOptions{
		RemoteAttempt: docs.RemoteAttemptOptions{
			Enabled:            true,
			APIBase:            options.remote.apiBase,
			AuthorizationToken: options.remote.authorizationToken,
		},
	}
}

func (options documentationOptions) RemoteDocumentationEnabled() bool {
	return options.remote.enabled
}

type githubTokenResolver interface {
	Resolve() (string, error)
}

type environmentGitHubTokenResolver struct {
	primaryEnv  string
	fallbackEnv string
}

func newEnvironmentGitHubTokenResolver(primaryEnv string, fallbackEnv string) environmentGitHubTokenResolver {
	return environmentGitHubTokenResolver{
		primaryEnv:  primaryEnv,
		fallbackEnv: fallbackEnv,
	}
}

func (resolver environmentGitHubTokenResolver) Resolve() (string, error) {
	primary := strings.TrimSpace(os.Getenv(resolver.primaryEnv))
	if primary != "" {
		return primary, nil
	}
	fallback := strings.TrimSpace(os.Getenv(resolver.fallbackEnv))
	if fallback != "" {
		return fallback, nil
	}
	return "", errGitHubTokenMissing
}
