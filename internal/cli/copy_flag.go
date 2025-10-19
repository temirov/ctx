package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

const (
	copyFlagTypeName            = "copy"
	invalidCopyFlagValueMessage = "invalid copy flag value '%s'"
)

var (
	trueCopyFlagLiterals = map[string]struct{}{
		"":     {},
		"true": {},
		"t":    {},
		"1":    {},
		"yes":  {},
		"y":    {},
	}
	falseCopyFlagLiterals = map[string]struct{}{
		"false": {},
		"f":     {},
		"0":     {},
		"no":    {},
		"n":     {},
	}
	copyFlagCommandNames = map[string]struct{}{
		"tree":      {},
		"t":         {},
		"content":   {},
		"c":         {},
		"callchain": {},
		"cc":        {},
		"doc":       {},
		"d":         {},
	}
)

func isCopyFlagCommand(argument string) bool {
	normalized := strings.ToLower(strings.TrimSpace(argument))
	_, known := copyFlagCommandNames[normalized]
	return known
}

func interpretCopyFlagLiteral(input string) (bool, bool) {
	normalized := strings.ToLower(strings.TrimSpace(input))
	if _, matches := trueCopyFlagLiterals[normalized]; matches {
		return true, true
	}
	if _, matches := falseCopyFlagLiterals[normalized]; matches {
		return false, true
	}
	return false, false
}

type copyFlagValue struct {
	target *bool
}

func (value *copyFlagValue) Set(input string) error {
	if value == nil || value.target == nil {
		return fmt.Errorf(invalidCopyFlagValueMessage, input)
	}
	booleanValue, ok := interpretCopyFlagLiteral(input)
	if !ok {
		return fmt.Errorf(invalidCopyFlagValueMessage, input)
	}
	*value.target = booleanValue
	return nil
}

func (value *copyFlagValue) String() string {
	if value == nil || value.target == nil {
		return "false"
	}
	if *value.target {
		return "true"
	}
	return "false"
}

func (value *copyFlagValue) Type() string {
	return copyFlagTypeName
}

func registerCopyFlag(flagSet *pflag.FlagSet, target *bool) {
	if flagSet == nil || target == nil {
		return
	}
	*target = false
	flagSet.Var(&copyFlagValue{target: target}, copyFlagName, copyFlagDescription)
	if lookup := flagSet.Lookup(copyFlagName); lookup != nil {
		lookup.NoOptDefVal = "true"
	}
}

func normalizeCopyFlagArguments(arguments []string) []string {
	if len(arguments) == 0 {
		return arguments
	}
	normalized := make([]string, 0, len(arguments))
	index := 0
	commandContext := false
	positionalOnly := false
	for index < len(arguments) {
		if positionalOnly {
			normalized = append(normalized, arguments[index:]...)
			break
		}
		current := arguments[index]
		if current == "--" {
			normalized = append(normalized, current)
			commandContext = true
			positionalOnly = true
			index++
			continue
		}
		if current == "--"+copyFlagName {
			nextIndex := index + 1
			if nextIndex >= len(arguments) || strings.HasPrefix(arguments[nextIndex], "-") {
				normalized = append(normalized, fmt.Sprintf("--%s=true", copyFlagName))
				index++
				continue
			}
			nextValue := arguments[nextIndex]
			if booleanValue, ok := interpretCopyFlagLiteral(nextValue); ok {
				normalized = append(normalized, fmt.Sprintf("--%s=%t", copyFlagName, booleanValue))
				index += 2
				continue
			}
			if commandContext || isCopyFlagCommand(nextValue) {
				normalized = append(normalized, current)
				index++
				continue
			}
			normalized = append(normalized, fmt.Sprintf("--%s=%s", copyFlagName, nextValue))
			index += 2
			continue
		}
		normalized = append(normalized, current)
		if !commandContext && !strings.HasPrefix(current, "-") && isCopyFlagCommand(current) {
			commandContext = true
		}
		index++
	}
	return normalized
}
