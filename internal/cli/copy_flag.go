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

type copyFlagValue struct {
	target *bool
}

func (value *copyFlagValue) Set(input string) error {
	if value == nil || value.target == nil {
		return fmt.Errorf(invalidCopyFlagValueMessage, input)
	}
	normalized := strings.ToLower(strings.TrimSpace(input))
	switch normalized {
	case "", "true", "t", "1", "yes", "y":
		*value.target = true
		return nil
	case "false", "f", "0", "no", "n":
		*value.target = false
		return nil
	default:
		return fmt.Errorf(invalidCopyFlagValueMessage, input)
	}
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
	for index < len(arguments) {
		current := arguments[index]
		if current == "--"+copyFlagName {
			nextIndex := index + 1
			if nextIndex >= len(arguments) || strings.HasPrefix(arguments[nextIndex], "-") {
				normalized = append(normalized, fmt.Sprintf("--%s=true", copyFlagName))
				index++
				continue
			}
			nextValue := strings.ToLower(strings.TrimSpace(arguments[nextIndex]))
			switch nextValue {
			case "true", "t", "1", "yes", "y":
				normalized = append(normalized, fmt.Sprintf("--%s=true", copyFlagName))
				index += 2
				continue
			case "false", "f", "0", "no", "n":
				normalized = append(normalized, fmt.Sprintf("--%s=false", copyFlagName))
				index += 2
				continue
			}
			normalized = append(normalized, fmt.Sprintf("--%s=%s", copyFlagName, nextValue))
			index += 2
			continue
		}
		normalized = append(normalized, current)
		index++
	}
	return normalized
}
