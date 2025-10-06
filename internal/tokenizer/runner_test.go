package tokenizer

import "testing"

func TestParseHelperTokenOutputLastLineInteger(t *testing.T) {
	count, err := parseHelperTokenOutput("123\n")
	if err != nil {
		t.Fatalf("parseHelperTokenOutput error: %v", err)
	}
	if count != 123 {
		t.Fatalf("expected 123 tokens, got %d", count)
	}
}

func TestParseHelperTokenOutputIgnoresPrefixedNoise(t *testing.T) {
	output := "Installed 14 packages in 20ms\n567\n"
	count, err := parseHelperTokenOutput(output)
	if err != nil {
		t.Fatalf("parseHelperTokenOutput error: %v", err)
	}
	if count != 567 {
		t.Fatalf("expected 567 tokens, got %d", count)
	}
}

func TestParseHelperTokenOutputEmpty(t *testing.T) {
	_, err := parseHelperTokenOutput("   \n  \n")
	if err == nil {
		t.Fatalf("expected error for empty output")
	}
	if err.Error() != "uv helper returned empty output" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestParseHelperTokenOutputInvalid(t *testing.T) {
	_, err := parseHelperTokenOutput("installed successfully\nno count")
	if err == nil {
		t.Fatalf("expected error for invalid output")
	}
	if err.Error() != "unexpected uv helper output: \"installed successfully\\nno count\"" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestSanitizeHelperOutputStripsInstalledNoise(t *testing.T) {
	input := "Installed 16 packages in 26ms\nModel not found"
	result := sanitizeHelperOutput(input)
	if result != "Model not found" {
		t.Fatalf("expected sanitized output to keep only model message, got %q", result)
	}
}

func TestSanitizeHelperOutputHandlesWhitespace(t *testing.T) {
	input := "\n\t Installed 20 packages in 30ms \n\nActual error line \n"
	result := sanitizeHelperOutput(input)
	if result != "Actual error line" {
		t.Fatalf("unexpected sanitized output: %q", result)
	}
}

func TestSanitizeHelperOutputReturnsEmptyWhenOnlyNoise(t *testing.T) {
	input := "Installed 5 packages in 5ms\n"
	result := sanitizeHelperOutput(input)
	if result != "" {
		t.Fatalf("expected empty sanitized output, got %q", result)
	}
}
