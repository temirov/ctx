package cli

import (
	"errors"
	"testing"

	"github.com/tyemirov/ctx/internal/types"
)

type staticTokenResolver struct {
	token string
	err   error
}

func (resolver staticTokenResolver) Resolve() (string, error) {
	if resolver.err != nil {
		return "", resolver.err
	}
	return resolver.token, nil
}

func TestDocumentationOptionsRemoteRequiresFullMode(t *testing.T) {
	_, err := newDocumentationOptions(documentationOptionsParameters{
		Mode:          types.DocumentationModeRelevant,
		RemoteEnabled: true,
		TokenResolver: staticTokenResolver{token: "token"},
	})
	if err == nil {
		t.Fatalf("expected error when remote documentation enabled with non-full mode")
	}
}

func TestDocumentationOptionsRemoteRequiresToken(t *testing.T) {
	_, err := newDocumentationOptions(documentationOptionsParameters{
		Mode:          types.DocumentationModeFull,
		RemoteEnabled: true,
		TokenResolver: staticTokenResolver{err: errGitHubTokenMissing},
	})
	if err == nil || !errors.Is(err, errGitHubTokenMissing) {
		t.Fatalf("expected missing token error, got %v", err)
	}
}

func TestDocumentationOptionsPropagatesRemoteSettings(t *testing.T) {
	options, err := newDocumentationOptions(documentationOptionsParameters{
		Mode:          types.DocumentationModeFull,
		RemoteEnabled: true,
		RemoteAPIBase: " https://example.com/api ",
		TokenResolver: staticTokenResolver{token: "secret"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	collectorOptions := options.CollectorOptions()
	if !collectorOptions.RemoteAttempt.Enabled {
		t.Fatalf("expected remote attempt enabled")
	}
	if collectorOptions.RemoteAttempt.APIBase != "https://example.com/api" {
		t.Fatalf("unexpected api base %q", collectorOptions.RemoteAttempt.APIBase)
	}
	if collectorOptions.RemoteAttempt.AuthorizationToken != "secret" {
		t.Fatalf("unexpected token %q", collectorOptions.RemoteAttempt.AuthorizationToken)
	}
}

func TestDocumentationOptionsAllowsMissingTokenWhenRemoteDisabled(t *testing.T) {
	options, err := newDocumentationOptions(documentationOptionsParameters{
		Mode:          types.DocumentationModeRelevant,
		RemoteEnabled: false,
		TokenResolver: staticTokenResolver{err: errGitHubTokenMissing},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.AuthorizationToken() != "" {
		t.Fatalf("expected empty token")
	}
	if options.Mode() != types.DocumentationModeRelevant {
		t.Fatalf("unexpected mode %s", options.Mode())
	}
}

func TestEnvironmentGitHubTokenResolverPrefersPrimary(t *testing.T) {
	resolver := newEnvironmentGitHubTokenResolver(githubTokenEnvPrimary, githubTokenEnvFallback)
	t.Setenv(githubTokenEnvPrimary, " primary-token ")
	t.Setenv(githubTokenEnvFallback, " fallback-token ")
	token, err := resolver.Resolve()
	if err != nil {
		t.Fatalf("unexpected error resolving token: %v", err)
	}
	if token != "primary-token" {
		t.Fatalf("expected trimmed primary token, got %q", token)
	}
}

func TestEnvironmentGitHubTokenResolverFallsBack(t *testing.T) {
	resolver := newEnvironmentGitHubTokenResolver(githubTokenEnvPrimary, githubTokenEnvFallback)
	t.Setenv(githubTokenEnvPrimary, "   ")
	t.Setenv(githubTokenEnvFallback, " fallback-token ")
	token, err := resolver.Resolve()
	if err != nil {
		t.Fatalf("unexpected error resolving token: %v", err)
	}
	if token != "fallback-token" {
		t.Fatalf("expected fallback token, got %q", token)
	}
}

func TestEnvironmentGitHubTokenResolverSignalsMissingToken(t *testing.T) {
	resolver := newEnvironmentGitHubTokenResolver(githubTokenEnvPrimary, githubTokenEnvFallback)
	t.Setenv(githubTokenEnvPrimary, "")
	t.Setenv(githubTokenEnvFallback, "")
	_, err := resolver.Resolve()
	if err == nil || !errors.Is(err, errGitHubTokenMissing) {
		t.Fatalf("expected missing token error, got %v", err)
	}
}
