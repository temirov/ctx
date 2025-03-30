package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/temirov/content/commands"
	"github.com/temirov/content/config"
	"github.com/temirov/content/utils"
)

// printUsage displays the command-line usage instructions and exits.
func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  content <tree|t|content|c> [directory1] [directory2] ... [-e|--e exclusion_folder] [--no-gitignore] [--no-ignore]")
	os.Exit(1)
}

func main() {
	commandName, directoryPaths, exclusionFolder, useGitignore, useIgnoreFile := parseArgsOrExit()
	err := runMultiDirectoryContentTool(commandName, directoryPaths, exclusionFolder, useGitignore, useIgnoreFile)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// parseArgsOrExit parses the command-line arguments.
// It identifies the command, collects directory paths, and parses flags.
// It returns the command name, a slice of directory paths, the exclusion folder,
// and boolean flags indicating whether to use .gitignore and .ignore files.
// Exits with usage information on error.
func parseArgsOrExit() (string, []string, string, bool, bool) {
	if len(os.Args) < 2 {
		printUsage()
	}

	rawCommand := os.Args[1]
	var commandName string
	switch rawCommand {
	case "tree", "t":
		commandName = "tree"
	case "content", "c":
		commandName = "content"
	default:
		fmt.Printf("Invalid command: %s\n", rawCommand)
		printUsage()
	}

	var directoryPaths []string
	exclusionFolder := ""
	useGitignore := true
	useIgnoreFile := true

	args := os.Args[2:]
	index := 0
	parsingFlags := false

	for index < len(args) {
		currentArg := args[index]

		isFlag := strings.HasPrefix(currentArg, "-")

		if isFlag {
			parsingFlags = true
			switch currentArg {
			case "-e", "--e":
				if index+1 >= len(args) {
					fmt.Println("Error: Missing exclusion folder value after -e/--e")
					printUsage()
				}
				exclusionFolder = args[index+1]
				index += 2
			case "--no-gitignore":
				useGitignore = false
				index++
			case "--no-ignore":
				useIgnoreFile = false
				index++
			default:
				fmt.Printf("Error: Unknown flag or misplaced argument: %s\n", currentArg)
				printUsage()
			}
		} else {
			if parsingFlags {
				fmt.Printf("Error: Positional argument '%s' found after flags.\n", currentArg)
				printUsage()
			}
			directoryPaths = append(directoryPaths, currentArg)
			index++
		}
	}

	if len(directoryPaths) == 0 {
		directoryPaths = []string{"."}
	}

	return commandName, directoryPaths, exclusionFolder, useGitignore, useIgnoreFile
}

// runMultiDirectoryContentTool orchestrates the execution for multiple directories.
// It validates directories, loads specific ignore patterns for each,
// and calls the appropriate command function.
func runMultiDirectoryContentTool(commandName string, inputDirectoryPaths []string, exclusionFolder string, useGitignore bool, useIgnoreFile bool) error {
	validatedAbsolutePaths, validationErr := validateAndDeduplicatePaths(inputDirectoryPaths)
	if validationErr != nil {
		return validationErr
	}

	var overallError error

	for _, absoluteDirectoryPath := range validatedAbsolutePaths {
		ignorePatterns, loadErr := loadIgnorePatternsForDirectory(absoluteDirectoryPath, exclusionFolder, useGitignore, useIgnoreFile)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Skipping directory %s due to error loading ignore patterns: %v\n", absoluteDirectoryPath, loadErr)
			if overallError == nil {
				overallError = loadErr
			}
			continue
		}

		var commandErr error
		switch commandName {
		case "tree":
			fmt.Printf("\n--- Directory Tree: %s ---\n", absoluteDirectoryPath)
			commandErr = commands.TreeCommand(absoluteDirectoryPath, ignorePatterns)
		case "content":
			commandErr = commands.ContentCommand(absoluteDirectoryPath, ignorePatterns)
		default:
			commandErr = fmt.Errorf("internal error: unhandled command '%s'", commandName)
		}

		if commandErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error processing directory %s: %v\n", absoluteDirectoryPath, commandErr)
			if overallError == nil {
				overallError = commandErr
			}
		}
	}

	return overallError
}

// validateAndDeduplicatePaths converts input paths to absolute paths,
// validates they are directories, and removes duplicates.
func validateAndDeduplicatePaths(inputPaths []string) ([]string, error) {
	uniqueAbsolutePaths := make(map[string]struct{})
	var finalPaths []string

	for _, inputPath := range inputPaths {
		absolutePath, absErr := filepath.Abs(inputPath)
		if absErr != nil {
			return nil, fmt.Errorf("error getting absolute path for '%s': %w", inputPath, absErr)
		}

		cleanPath := filepath.Clean(absolutePath)

		if _, exists := uniqueAbsolutePaths[cleanPath]; exists {
			continue
		}

		if !utils.IsDirectory(cleanPath) {
			return nil, fmt.Errorf("error: '%s' (resolved to '%s') is not a valid directory", inputPath, cleanPath)
		}

		uniqueAbsolutePaths[cleanPath] = struct{}{}
		finalPaths = append(finalPaths, cleanPath)
	}

	if len(finalPaths) == 0 {
		return nil, fmt.Errorf("error: no valid directories found to process")
	}

	return finalPaths, nil
}

// loadIgnorePatternsForDirectory loads .ignore and .gitignore patterns for a specific directory.
// It respects the useGitignore and useIgnoreFile flags and adds the exclusionFolder pattern.
func loadIgnorePatternsForDirectory(absoluteDirectoryPath string, exclusionFolder string, useGitignore bool, useIgnoreFile bool) ([]string, error) {
	var ignorePatterns []string

	if useIgnoreFile {
		ignoreFilePath := filepath.Join(absoluteDirectoryPath, ".ignore")
		loadedIgnorePatterns, loadError := config.LoadContentIgnore(ignoreFilePath)
		if loadError != nil && !os.IsNotExist(loadError) {
			return nil, fmt.Errorf("error loading .ignore from %s: %w", absoluteDirectoryPath, loadError)
		}
		ignorePatterns = append(ignorePatterns, loadedIgnorePatterns...)
	}

	if useGitignore {
		gitIgnoreFilePath := filepath.Join(absoluteDirectoryPath, ".gitignore")
		loadedGitignorePatterns, gitignoreErr := config.LoadContentIgnore(gitIgnoreFilePath)
		if gitignoreErr != nil && !os.IsNotExist(gitignoreErr) {
			return nil, fmt.Errorf("error loading .gitignore from %s: %w", absoluteDirectoryPath, gitignoreErr)
		}
		ignorePatterns = append(ignorePatterns, loadedGitignorePatterns...)
	}

	ignorePatterns = deduplicatePatterns(ignorePatterns)

	trimmedExclusion := strings.TrimSpace(exclusionFolder)
	if trimmedExclusion != "" {
		normalizedExclusion := strings.TrimSuffix(trimmedExclusion, "/")
		ignorePatterns = append(ignorePatterns, "EXCL:"+normalizedExclusion)
	}

	return ignorePatterns, nil
}

// deduplicatePatterns removes duplicate patterns from a slice.
func deduplicatePatterns(patterns []string) []string {
	patternSet := make(map[string]struct{})
	result := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		if _, exists := patternSet[pattern]; !exists {
			patternSet[pattern] = struct{}{}
			result = append(result, pattern)
		}
	}
	return result
}
