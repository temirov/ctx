package tokenizer

import "testing"

type testCounter struct{}

func (testCounter) Name() string { return "stub" }

func (testCounter) CountString(input string) (int, error) { return len([]rune(input)), nil }

func TestCountBytesText(t *testing.T) {
	result, err := CountBytes(testCounter{}, []byte("hello"))
	if err != nil {
		t.Fatalf("CountBytes error: %v", err)
	}
	if !result.Counted {
		t.Fatalf("expected counted result")
	}
	if result.Tokens != len([]rune("hello")) {
		t.Fatalf("expected %d tokens, got %d", len([]rune("hello")), result.Tokens)
	}
}

func TestCountBytesBinary(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02}
	result, err := CountBytes(testCounter{}, data)
	if err != nil {
		t.Fatalf("CountBytes error: %v", err)
	}
	if result.Counted {
		t.Fatalf("expected binary data to be skipped")
	}
}

func TestNewCounterDefault(t *testing.T) {
	counter, model, err := NewCounter(Config{Model: "gpt-4o"})
	if err != nil {
		t.Fatalf("NewCounter error: %v", err)
	}
	if counter == nil {
		t.Fatalf("expected non-nil counter")
	}
	if model != "gpt-4o" {
		t.Fatalf("expected model gpt-4o, got %q", model)
	}
	tokens, err := counter.CountString("hello world")
	if err != nil {
		t.Fatalf("CountString error: %v", err)
	}
	if tokens <= 0 {
		t.Fatalf("expected positive token count, got %d", tokens)
	}
}
