package tokenizer

import (
	"errors"

	"github.com/pkoukk/tiktoken-go"
)

type openAICounter struct {
	encoding *tiktoken.Tiktoken
	name     string
}

func (counter openAICounter) Name() string {
	return counter.name
}

func (counter openAICounter) CountString(input string) (int, error) {
	if counter.encoding == nil {
		return 0, errors.New("nil tiktoken encoder")
	}
	tokenIDs := counter.encoding.Encode(input, nil, nil)
	return len(tokenIDs), nil
}
