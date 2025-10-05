package tokenizer

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkoukk/tiktoken-go"
)

// Counter estimates token counts for text content.
type Counter interface {
	Name() string
	CountString(input string) (int, error)
}

// Config captures tokenizer selection parameters provided by the CLI.
type Config struct {
	Model                  string
	PythonExecutable       string
	HelpersDir             string
	SentencePieceModelPath string
	WorkingDirectory       string
	Timeout                time.Duration
}

const (
	defaultModel          = "gpt-4o"
	defaultEncodingName   = "cl100k_base"
	pythonExecutableGuess = "python3"
	defaultPythonTimeout  = 120 * time.Second
)

// NewCounter returns a Counter implementation for the requested model.
func NewCounter(cfg Config) (Counter, error) {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultModel
	}
	lowerModel := strings.ToLower(model)

	if isOpenAIModel(lowerModel) {
		encoding, err := tiktoken.EncodingForModel(lowerModel)
		if err == nil && encoding != nil {
			return openAICounter{encoding: encoding, name: lowerModel}, nil
		}
		fallback, fallbackErr := tiktoken.GetEncoding(defaultEncodingName)
		if fallbackErr != nil {
			return nil, fmt.Errorf("initialize fallback tokenizer: %w", fallbackErr)
		}
		return openAICounter{encoding: fallback, name: defaultEncodingName}, nil
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultPythonTimeout
	}

	helpersDir := resolvePath(cfg.WorkingDirectory, strings.TrimSpace(cfg.HelpersDir))
	pythonExecutable := strings.TrimSpace(cfg.PythonExecutable)
	if pythonExecutable == "" {
		pythonExecutable = pythonExecutableGuess
	}

	switch {
	case strings.HasPrefix(lowerModel, "claude-"):
		directory, err := materializeHelperScripts(helpersDir)
		if err != nil {
			return nil, err
		}
		scriptPath := filepath.Join(directory, anthropicScriptName)
		return pythonCounter{
			executable: pythonExecutable,
			scriptPath: scriptPath,
			args:       []string{"--model", lowerModel},
			helperName: "anthropic_tokenizer",
			timeout:    timeout,
		}, nil
	case strings.HasPrefix(lowerModel, "llama-"):
		spModelPath := resolvePath(cfg.WorkingDirectory, strings.TrimSpace(cfg.SentencePieceModelPath))
		if spModelPath == "" {
			return nil, errors.New("llama model requires --spm-model path to SentencePiece tokenizer.model")
		}
		directory, err := materializeHelperScripts(helpersDir)
		if err != nil {
			return nil, err
		}
		scriptPath := filepath.Join(directory, llamaScriptName)
		return pythonCounter{
			executable: pythonExecutable,
			scriptPath: scriptPath,
			args:       []string{"--spm-model", spModelPath},
			helperName: "sentencepiece",
			timeout:    timeout,
		}, nil
	default:
		encoding, err := tiktoken.GetEncoding(defaultEncodingName)
		if err != nil {
			return nil, fmt.Errorf("initialize default tokenizer: %w", err)
		}
		return openAICounter{encoding: encoding, name: defaultEncodingName}, nil
	}
}

func isOpenAIModel(model string) bool {
	prefixes := []string{
		"gpt-",
		"text-embedding",
		"davinci",
		"curie",
		"babbage",
		"ada",
		"code-",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}

func resolvePath(base string, path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) || base == "" {
		return path
	}
	return filepath.Join(base, path)
}
