package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	booleanFlagTypeName               = "bool"
	booleanFlagTrueLiteral            = "true"
	booleanFlagAcceptedValuesListing  = "true, false, yes, no, on, off, 1, 0"
	booleanFlagInvalidValueErrorLabel = "invalid boolean value"
)

var booleanFlagLiterals = map[string]bool{
	"true":  true,
	"t":     true,
	"1":     true,
	"yes":   true,
	"y":     true,
	"on":    true,
	"false": false,
	"f":     false,
	"0":     false,
	"no":    false,
	"n":     false,
	"off":   false,
}

type booleanFlagValue struct {
	target  *bool
	flagKey string
}

func (value *booleanFlagValue) Set(input string) error {
	if value == nil || value.target == nil {
		return fmt.Errorf("%s %q for flag %q", booleanFlagInvalidValueErrorLabel, input, value.flagKey)
	}
	normalized := strings.ToLower(strings.TrimSpace(input))
	if normalized == "" {
		normalized = booleanFlagTrueLiteral
	}
	parsed, ok := booleanFlagLiterals[normalized]
	if !ok {
		return fmt.Errorf("%s %q for --%s; accepted values: %s", booleanFlagInvalidValueErrorLabel, input, value.flagKey, booleanFlagAcceptedValuesListing)
	}
	*value.target = parsed
	return nil
}

func (value *booleanFlagValue) String() string {
	if value == nil || value.target == nil {
		return booleanFlagTrueLiteral
	}
	return strconv.FormatBool(*value.target)
}

func (value *booleanFlagValue) Type() string {
	return booleanFlagTypeName
}

func registerBooleanFlag(flagSet *pflag.FlagSet, target *bool, name string, defaultValue bool, usage string) {
	if flagSet == nil || target == nil {
		return
	}
	*target = defaultValue
	flagValue := &booleanFlagValue{
		target:  target,
		flagKey: name,
	}
	flagSet.Var(flagValue, name, usage)
	if lookup := flagSet.Lookup(name); lookup != nil {
		lookup.DefValue = strconv.FormatBool(defaultValue)
		lookup.NoOptDefVal = booleanFlagTrueLiteral
	}
}

func normalizeBooleanFlagArguments(command *cobra.Command, arguments []string) []string {
	if command == nil || len(arguments) == 0 {
		return arguments
	}
	booleanFlags := map[string]struct{}{}
	collectBooleanFlagNames(command, booleanFlags)
	if len(booleanFlags) == 0 {
		return arguments
	}
	normalized := make([]string, 0, len(arguments))
	index := 0
	for index < len(arguments) {
		currentArgument := arguments[index]
		if currentArgument == "--" {
			normalized = append(normalized, arguments[index:]...)
			break
		}
		if strings.HasPrefix(currentArgument, "--") && !strings.Contains(currentArgument, "=") {
			flagName := strings.TrimPrefix(currentArgument, "--")
			if _, exists := booleanFlags[flagName]; exists && index+1 < len(arguments) {
				nextArgument := arguments[index+1]
				if !strings.HasPrefix(nextArgument, "-") {
					literal := strings.ToLower(strings.TrimSpace(nextArgument))
					if _, valid := booleanFlagLiterals[literal]; valid {
						normalized = append(normalized, fmt.Sprintf("--%s=%s", flagName, nextArgument))
						index += 2
						continue
					}
				}
			}
		}
		normalized = append(normalized, currentArgument)
		index++
	}
	return normalized
}

func collectBooleanFlagNames(command *cobra.Command, target map[string]struct{}) {
	if command == nil || target == nil {
		return
	}
	visit := func(flagSet *pflag.FlagSet) {
		if flagSet == nil {
			return
		}
		flagSet.VisitAll(func(flag *pflag.Flag) {
			if flag == nil || flag.Value == nil {
				return
			}
			if flag.Value.Type() == booleanFlagTypeName {
				target[flag.Name] = struct{}{}
			}
		})
	}
	visit(command.PersistentFlags())
	visit(command.Flags())
	for _, child := range command.Commands() {
		collectBooleanFlagNames(child, target)
	}
}
