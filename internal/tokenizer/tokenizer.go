package tokenizer

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
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
	Model            string
	WorkingDirectory string
	Timeout          time.Duration
}

const (
	defaultModel        = "gpt-4o"
	defaultEncodingName = "cl100k_base"
	defaultUVTimeout    = 120 * time.Second
)

// NewCounter returns a Counter implementation for the requested model.
func NewCounter(cfg Config) (Counter, string, error) {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultModel
	}
	lowerModel := strings.ToLower(model)

	if isOpenAIModel(lowerModel) {
		encoding, err := tiktoken.EncodingForModel(lowerModel)
		if err == nil && encoding != nil {
			return openAICounter{encoding: encoding, name: lowerModel}, model, nil
		}
		fallback, fallbackErr := tiktoken.GetEncoding(defaultEncodingName)
		if fallbackErr != nil {
			return nil, "", fmt.Errorf("initialize fallback tokenizer: %w", fallbackErr)
		}
		return openAICounter{encoding: fallback, name: defaultEncodingName}, defaultEncodingName, nil
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultUVTimeout
	}

	uvExecutable, detectErr := detectUVExecutable()
	if detectErr != nil {
		return nil, "", detectErr
	}

	switch {
	case strings.HasPrefix(lowerModel, "claude-"):
		directory, err := materializeHelperScripts("")
		if err != nil {
			return nil, "", err
		}
		scriptPath := filepath.Join(directory, anthropicScriptName)
		return scriptCounter{
			runner:     uvExecutable,
			scriptPath: scriptPath,
			args:       []string{"--model", model},
			helperName: "anthropic_tokenizer",
			timeout:    timeout,
		}, model, nil
	case strings.HasPrefix(lowerModel, "llama-"):
		spmModelPath := strings.TrimSpace(os.Getenv("CTX_SPM_MODEL"))
		if spmModelPath == "" {
			return nil, "", errors.New("llama models require CTX_SPM_MODEL to point to a SentencePiece tokenizer.model file")
		}
		spmModelPath = resolvePath(cfg.WorkingDirectory, spmModelPath)
		if _, err := os.Stat(spmModelPath); err != nil {
			return nil, "", fmt.Errorf("unable to access SentencePiece model %s: %w", spmModelPath, err)
		}
		directory, err := materializeHelperScripts("")
		if err != nil {
			return nil, "", err
		}
		scriptPath := filepath.Join(directory, llamaScriptName)
		return scriptCounter{
			runner:     uvExecutable,
			scriptPath: scriptPath,
			args:       []string{"--spm-model", spmModelPath},
			helperName: "sentencepiece",
			timeout:    timeout,
		}, model, nil
	default:
		encoding, err := tiktoken.GetEncoding(defaultEncodingName)
		if err != nil {
			return nil, "", fmt.Errorf("initialize default tokenizer: %w", err)
		}
		return openAICounter{encoding: encoding, name: defaultEncodingName}, defaultEncodingName, nil
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

func detectUVExecutable() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("CTX_UV")); explicit != "" {
		if path, err := exec.LookPath(explicit); err == nil {
			return path, nil
		}
		if _, err := os.Stat(explicit); err == nil {
			return explicit, nil
		}
		return "", fmt.Errorf("uv executable specified via CTX_UV (%s) not found", explicit)
	}
	if path, err := exec.LookPath("uv"); err == nil {
		return path, nil
	}
	return "", errors.New("uv executable not found; install uv from https://github.com/astral-sh/uv or expose it via CTX_UV")
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
