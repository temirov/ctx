package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/temirov/ctx/internal/utils"
)

// InitTarget identifies where configuration should be initialized.
type InitTarget string

const (
	// InitTargetLocal writes configuration into the working directory.
	InitTargetLocal InitTarget = "local"
	// InitTargetGlobal writes configuration into the global configuration directory.
	InitTargetGlobal InitTarget = "global"

	defaultConfigurationTemplate = `tree:
  format: json
  summary: true
  content: false
  tokens:
    enabled: false
    model: gpt-4o
  paths:
    exclude: []
    use_gitignore: true
    use_ignore: true
    include_git: false
content:
  format: json
  summary: true
  content: true
  documentation: false
  tokens:
    enabled: false
    model: gpt-4o
  paths:
    exclude: []
    use_gitignore: true
    use_ignore: true
    include_git: false
callchain:
  format: json
  depth: 1
  documentation: false
`
)

// InitOptions controls how configuration initialization behaves.
type InitOptions struct {
	Target           InitTarget
	Force            bool
	WorkingDirectory string
}

// InitializeConfiguration writes the default configuration to the requested target.
func InitializeConfiguration(options InitOptions) (string, error) {
	target := options.Target
	if target == "" {
		target = InitTargetLocal
	}
	var destinationPath string
	switch target {
	case InitTargetLocal:
		workingDirectory := options.WorkingDirectory
		if workingDirectory == "" {
			current, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("determine working directory for configuration: %w", err)
			}
			workingDirectory = current
		}
		destinationPath = filepath.Join(workingDirectory, utils.ConfigFileName)
	case InitTargetGlobal:
		homeDirectory, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory for configuration: %w", err)
		}
		configurationDirectory := filepath.Join(homeDirectory, utils.GlobalConfigDirectoryName)
		if err := os.MkdirAll(configurationDirectory, 0o755); err != nil {
			return "", fmt.Errorf("create configuration directory %s: %w", configurationDirectory, err)
		}
		destinationPath = filepath.Join(configurationDirectory, utils.ConfigFileName)
	default:
		return "", fmt.Errorf("unsupported init target %q", target)
	}

	if _, err := os.Stat(destinationPath); err == nil {
		if !options.Force {
			return "", fmt.Errorf("configuration file already exists at %s", destinationPath)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect configuration path %s: %w", destinationPath, err)
	}

	if err := os.WriteFile(destinationPath, []byte(defaultConfigurationTemplate), 0o600); err != nil {
		return "", fmt.Errorf("write configuration to %s: %w", destinationPath, err)
	}

	return destinationPath, nil
}
