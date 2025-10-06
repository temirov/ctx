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
	defaultModel         = "gpt-4o"
	defaultEncodingName  = "cl100k_base"
	defaultPythonTimeout = 120 * time.Second
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
		timeout = defaultPythonTimeout
	}

	pythonExecutable, detectErr := detectPythonExecutable()
	if detectErr != nil {
		return nil, "", detectErr
	}

	switch {
	case strings.HasPrefix(lowerModel, "claude-"):
		if err := ensurePythonModule(pythonExecutable, "anthropic_tokenizer"); err != nil {
			return nil, "", err
		}
		directory, err := materializeHelperScripts("")
		if err != nil {
			return nil, "", err
		}
		scriptPath := filepath.Join(directory, anthropicScriptName)
		return pythonCounter{
			executable: pythonExecutable,
			scriptPath: scriptPath,
			args:       []string{"--model", lowerModel},
			helperName: "anthropic_tokenizer",
			timeout:    timeout,
		}, model, nil
	case strings.HasPrefix(lowerModel, "llama-"):
		if err := ensurePythonModule(pythonExecutable, "sentencepiece"); err != nil {
			return nil, "", err
		}
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
		return pythonCounter{
			executable: pythonExecutable,
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

func detectPythonExecutable() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("CTX_PYTHON")); explicit != "" {
		if _, err := os.Stat(explicit); err == nil {
			if err := verifyPythonCompatibility(explicit); err != nil {
				return "", fmt.Errorf("python specified via CTX_PYTHON (%s) is not compatible: %w", explicit, err)
			}
			return explicit, nil
		}
		if path, err := exec.LookPath(explicit); err == nil {
			if err := verifyPythonCompatibility(path); err != nil {
				return "", fmt.Errorf("python specified via CTX_PYTHON (%s) is not compatible: %w", path, err)
			}
			return path, nil
		}
		return "", fmt.Errorf("python executable specified via CTX_PYTHON (%s) not found", explicit)
	}

	candidates := []string{"python3", "python"}
	for _, candidate := range candidates {
		path, err := exec.LookPath(candidate)
		if err != nil {
			continue
		}
		if err := verifyPythonCompatibility(path); err != nil {
			continue
		}
		return path, nil
	}
	return "", errors.New("python 3.8+ not found; install Python or set CTX_PYTHON to a compatible interpreter")
}

func verifyPythonCompatibility(pythonPath string) error {
	cmd := exec.Command(pythonPath, "-c", "import sys; sys.exit(0 if sys.version_info >= (3, 8) else 1)")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("python interpreter %s must be version 3.8 or newer", pythonPath)
	}
	return nil
}

func ensurePythonModule(pythonPath, module string) error {
	cmd := exec.Command(pythonPath, "-c", fmt.Sprintf("import %s", module))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("python module %s not available; install it in your environment", module)
	}
	return nil
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
