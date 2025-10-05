package tokenizer

import (
	"errors"
	"os"
	"unicode/utf8"

	"github.com/temirov/ctx/internal/utils"
)

// CountResult captures the outcome of counting a file or byte slice.
type CountResult struct {
	Tokens  int
	Counted bool
}

// CountBytes estimates tokens for the provided data using counter.
func CountBytes(counter Counter, data []byte) (CountResult, error) {
	if counter == nil {
		return CountResult{}, errors.New("nil tokenizer counter")
	}
	if len(data) == 0 {
		tokens, err := counter.CountString("")
		if err != nil {
			return CountResult{}, err
		}
		return CountResult{Tokens: tokens, Counted: true}, nil
	}
	if utils.IsBinary(data) {
		return CountResult{Counted: false}, nil
	}
	if !utf8.Valid(data) {
		return CountResult{Counted: false}, nil
	}
	tokens, err := counter.CountString(string(data))
	if err != nil {
		return CountResult{}, err
	}
	return CountResult{Tokens: tokens, Counted: true}, nil
}

// CountFile reads the file at path and estimates its token count.
func CountFile(counter Counter, path string) (CountResult, error) {
	if counter == nil {
		return CountResult{}, errors.New("nil tokenizer counter")
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		return CountResult{}, readErr
	}
	return CountBytes(counter, data)
}
