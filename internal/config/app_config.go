package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/temirov/ctx/internal/utils"
)

// LoadOptions controls how application configuration is discovered.
type LoadOptions struct {
	WorkingDirectory string
	ExplicitFilePath string
}

// ApplicationConfiguration holds command-specific configuration defaults.
type ApplicationConfiguration struct {
	Tree      StreamCommandConfiguration `mapstructure:"tree"`
	Content   StreamCommandConfiguration `mapstructure:"content"`
	CallChain CallChainConfiguration     `mapstructure:"callchain"`
}

// StreamCommandConfiguration defines options shared by tree and content commands.
type StreamCommandConfiguration struct {
	Format            string             `mapstructure:"format"`
	Summary           *bool              `mapstructure:"summary"`
	Documentation     *bool              `mapstructure:"documentation"`
	DocumentationMode string             `mapstructure:"documentation_mode"`
	IncludeContent    *bool              `mapstructure:"content"`
	DocsAttempt       *bool              `mapstructure:"docs_attempt"`
	Tokens            TokenConfiguration `mapstructure:"tokens"`
	Paths             PathConfiguration  `mapstructure:"paths"`
	Clipboard         *bool              `mapstructure:"clipboard"`
}

// TokenConfiguration controls token counting defaults.
type TokenConfiguration struct {
	Enabled *bool  `mapstructure:"enabled"`
	Model   string `mapstructure:"model"`
}

// PathConfiguration configures inclusion and exclusion rules for path traversal.
type PathConfiguration struct {
	Exclude       []string `mapstructure:"exclude"`
	UseGitignore  *bool    `mapstructure:"use_gitignore"`
	UseIgnoreFile *bool    `mapstructure:"use_ignore"`
	IncludeGit    *bool    `mapstructure:"include_git"`
}

// CallChainConfiguration defines defaults for the callchain command.
type CallChainConfiguration struct {
	Format            string `mapstructure:"format"`
	Depth             *int   `mapstructure:"depth"`
	Documentation     *bool  `mapstructure:"documentation"`
	DocumentationMode string `mapstructure:"documentation_mode"`
	DocsAttempt       *bool  `mapstructure:"docs_attempt"`
	Clipboard         *bool  `mapstructure:"clipboard"`
}

// LoadApplicationConfiguration loads configuration from global and local files.
func LoadApplicationConfiguration(options LoadOptions) (ApplicationConfiguration, error) {
	workingDirectory := options.WorkingDirectory
	if workingDirectory == "" {
		currentDirectory, err := os.Getwd()
		if err != nil {
			return ApplicationConfiguration{}, fmt.Errorf("determine working directory: %w", err)
		}
		workingDirectory = currentDirectory
	}

	var merged ApplicationConfiguration

	if homeDirectory, err := os.UserHomeDir(); err == nil && homeDirectory != "" {
		globalPath := filepath.Join(homeDirectory, utils.GlobalConfigDirectoryName, utils.ConfigFileName)
		globalConfig, loadErr := loadConfigurationFromPath(globalPath)
		if loadErr != nil {
			return ApplicationConfiguration{}, loadErr
		}
		merged = merged.Merge(globalConfig)
	}

	localPath, resolveErr := resolveLocalConfigPath(workingDirectory, options.ExplicitFilePath)
	if resolveErr != nil {
		return ApplicationConfiguration{}, resolveErr
	}
	if localPath != "" {
		localConfig, loadErr := loadConfigurationFromPath(localPath)
		if loadErr != nil {
			return ApplicationConfiguration{}, loadErr
		}
		merged = merged.Merge(localConfig)
	}

	merged.Tree.Paths.Exclude = utils.DeduplicatePatterns(merged.Tree.Paths.Exclude)
	merged.Content.Paths.Exclude = utils.DeduplicatePatterns(merged.Content.Paths.Exclude)

	return merged, nil
}

func resolveLocalConfigPath(workingDirectory, explicitPath string) (string, error) {
	if explicitPath != "" {
		if filepath.IsAbs(explicitPath) {
			return explicitPath, nil
		}
		if workingDirectory == "" {
			absolute, err := filepath.Abs(explicitPath)
			if err != nil {
				return "", fmt.Errorf("resolve configuration path %s: %w", explicitPath, err)
			}
			return absolute, nil
		}
		return filepath.Join(workingDirectory, explicitPath), nil
	}
	if workingDirectory == "" {
		return "", nil
	}
	return filepath.Join(workingDirectory, utils.ConfigFileName), nil
}

func loadConfigurationFromPath(path string) (ApplicationConfiguration, error) {
	if path == "" {
		return ApplicationConfiguration{}, nil
	}
	info, statErr := os.Stat(path)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return ApplicationConfiguration{}, nil
		}
		return ApplicationConfiguration{}, fmt.Errorf("stat configuration %s: %w", path, statErr)
	}
	if info.IsDir() {
		return ApplicationConfiguration{}, fmt.Errorf("configuration path %s is a directory", path)
	}

	reader := viper.New()
	reader.SetConfigFile(path)
	if readErr := reader.ReadInConfig(); readErr != nil {
		return ApplicationConfiguration{}, fmt.Errorf("read configuration from %s: %w", path, readErr)
	}
	var config ApplicationConfiguration
	if decodeErr := reader.Unmarshal(&config); decodeErr != nil {
		return ApplicationConfiguration{}, fmt.Errorf("decode configuration from %s: %w", path, decodeErr)
	}
	return config, nil
}

// Merge overlays override onto the receiver returning the combined configuration.
func (config ApplicationConfiguration) Merge(override ApplicationConfiguration) ApplicationConfiguration {
	result := config
	result.Tree = result.Tree.merge(override.Tree)
	result.Content = result.Content.merge(override.Content)
	result.CallChain = result.CallChain.merge(override.CallChain)
	return result
}

func (config StreamCommandConfiguration) merge(override StreamCommandConfiguration) StreamCommandConfiguration {
	result := config
	if override.Format != "" {
		result.Format = override.Format
	}
	if override.Summary != nil {
		result.Summary = cloneBool(override.Summary)
	}
	if override.Documentation != nil {
		result.Documentation = cloneBool(override.Documentation)
	}
	if override.DocumentationMode != "" {
		result.DocumentationMode = override.DocumentationMode
	}
	if override.DocumentationMode != "" {
		result.DocumentationMode = override.DocumentationMode
	}
	if override.IncludeContent != nil {
		result.IncludeContent = cloneBool(override.IncludeContent)
	}
	if override.DocsAttempt != nil {
		result.DocsAttempt = cloneBool(override.DocsAttempt)
	}
	result.Tokens = result.Tokens.merge(override.Tokens)
	result.Paths = result.Paths.merge(override.Paths)
	if override.Clipboard != nil {
		result.Clipboard = cloneBool(override.Clipboard)
	}
	return result
}

func (config TokenConfiguration) merge(override TokenConfiguration) TokenConfiguration {
	result := config
	if override.Enabled != nil {
		result.Enabled = cloneBool(override.Enabled)
	}
	if override.Model != "" {
		result.Model = override.Model
	}
	return result
}

func (config PathConfiguration) merge(override PathConfiguration) PathConfiguration {
	result := config
	if len(override.Exclude) > 0 {
		result.Exclude = append([]string{}, utils.DeduplicatePatterns(override.Exclude)...)
	}
	if override.UseGitignore != nil {
		result.UseGitignore = cloneBool(override.UseGitignore)
	}
	if override.UseIgnoreFile != nil {
		result.UseIgnoreFile = cloneBool(override.UseIgnoreFile)
	}
	if override.IncludeGit != nil {
		result.IncludeGit = cloneBool(override.IncludeGit)
	}
	return result
}

func (config CallChainConfiguration) merge(override CallChainConfiguration) CallChainConfiguration {
	result := config
	if override.Format != "" {
		result.Format = override.Format
	}
	if override.Depth != nil {
		result.Depth = cloneInt(override.Depth)
	}
	if override.Documentation != nil {
		result.Documentation = cloneBool(override.Documentation)
	}
	if override.DocumentationMode != "" {
		result.DocumentationMode = override.DocumentationMode
	}
	if override.DocsAttempt != nil {
		result.DocsAttempt = cloneBool(override.DocsAttempt)
	}
	if override.Clipboard != nil {
		result.Clipboard = cloneBool(override.Clipboard)
	}
	return result
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
