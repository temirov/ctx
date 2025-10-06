package tokenizer

import "strings"

// ProgressLabel returns a human-friendly label for progress messages when the
// counter relies on external helpers. The boolean indicates whether progress
// should be reported for the counter.
func ProgressLabel(counter Counter, model string) (string, bool) {
	if counter == nil {
		return "", false
	}

	switch counter.(type) {
	case scriptCounter, *scriptCounter:
		label := strings.TrimSpace(model)
		if label == "" {
			label = strings.TrimSpace(counter.Name())
		}
		if label == "" {
			label = "helper"
		}
		return label, true
	default:
		return "", false
	}
}
