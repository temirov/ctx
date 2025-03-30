package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/temirov/content/commands"
	"github.com/temirov/content/config"
)

// ValidatedPath stores information about a resolved and validated input path.
type ValidatedPath struct {
	AbsolutePath string
	IsDir        bool
}

// printUsage displays the command-line usage instructions and exits.
func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  content <tree|t|content|c> [path1] [path2] ... [-e|--e exclusion_folder] [--no-gitignore] [--no-ignore]")
	fmt.Println("\nPaths can be files or directories.")
	os.Exit(1)
}

func main() {
	commandName, inputPaths, exclusionFolder, useGitignore, useIgnoreFile := parseArgsOrExit()
	err := runContentTool(commandName, inputPaths, exclusionFolder, useGitignore, useIgnoreFile)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// parseArgsOrExit parses the command-line arguments.
// It identifies the command, collects input paths (files or directories), and parses flags.
// Returns the command name, a slice of input paths, the exclusion folder,
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

	var inputPaths []string
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
			inputPaths = append(inputPaths, currentArg)
			index++
		}
	}

	if len(inputPaths) == 0 {
		inputPaths = []string{"."}
	}

	return commandName, inputPaths, exclusionFolder, useGitignore, useIgnoreFile
}

// runContentTool orchestrates the execution for multiple input paths (files or directories).
// It validates paths, determines their type, loads ignores for directories,
// and calls the appropriate command function or prints file content directly.
func runContentTool(commandName string, inputPaths []string, exclusionFolder string, useGitignore bool, useIgnoreFile bool) error {
	validatedPaths, validationErr := resolveAndValidatePaths(inputPaths)
	if validationErr != nil {
		return validationErr
	}

	var overallError error

	for _, pathInfo := range validatedPaths {
		var processingErr error

		switch commandName {
		case "tree":
			if pathInfo.IsDir {
				ignorePatterns, loadErr := loadIgnorePatternsForDirectory(pathInfo.AbsolutePath, exclusionFolder, useGitignore, useIgnoreFile)
				if loadErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: Skipping directory %s due to error loading ignore patterns: %v\n", pathInfo.AbsolutePath, loadErr)
					if overallError == nil {
						overallError = loadErr
					}
					continue
				}
				fmt.Printf("\n--- Directory Tree: %s ---\n", pathInfo.AbsolutePath)
				processingErr = commands.TreeCommand(pathInfo.AbsolutePath, ignorePatterns)
			} else {
				fmt.Printf("[File] %s\n", pathInfo.AbsolutePath)
				processingErr = nil
			}
		case "content":
			if pathInfo.IsDir {
				ignorePatterns, loadErr := loadIgnorePatternsForDirectory(pathInfo.AbsolutePath, exclusionFolder, useGitignore, useIgnoreFile)
				if loadErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: Skipping directory %s due to error loading ignore patterns: %v\n", pathInfo.AbsolutePath, loadErr)
					if overallError == nil {
						overallError = loadErr
					}
					continue
				}
				processingErr = commands.ContentCommand(pathInfo.AbsolutePath, ignorePatterns)
			} else {
				processingErr = printSingleFileContent(pathInfo.AbsolutePath)
			}
		default:
			processingErr = fmt.Errorf("internal error: unhandled command '%s'", commandName)
		}

		if processingErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error processing path %s: %v\n", pathInfo.AbsolutePath, processingErr)
			if overallError == nil {
				overallError = processingErr
			}
		}
	}

	return overallError
}

// resolveAndValidatePaths converts input paths to absolute paths, checks existence,
// determines if they are files or directories, and removes duplicates.
func resolveAndValidatePaths(inputPaths []string) ([]ValidatedPath, error) {
	uniquePaths := make(map[string]struct{})
	var validatedPaths []ValidatedPath

	for _, inputPath := range inputPaths {
		absolutePath, absErr := filepath.Abs(inputPath)
		if absErr != nil {
			return nil, fmt.Errorf("error getting absolute path for '%s': %w", inputPath, absErr)
		}

		cleanPath := filepath.Clean(absolutePath)

		if _, exists := uniquePaths[cleanPath]; exists {
			continue
		}

		fileInfo, statErr := os.Stat(cleanPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return nil, fmt.Errorf("error: path '%s' (resolved to '%s') does not exist", inputPath, cleanPath)
			}
			return nil, fmt.Errorf("error stating path '%s' (resolved to '%s'): %w", inputPath, cleanPath, statErr)
		}

		uniquePaths[cleanPath] = struct{}{}
		validatedPaths = append(validatedPaths, ValidatedPath{
			AbsolutePath: cleanPath,
			IsDir:        fileInfo.IsDir(),
		})
	}

	if len(validatedPaths) == 0 {
		return nil, fmt.Errorf("error: no valid paths found to process")
	}

	return validatedPaths, nil
}

// loadIgnorePatternsForDirectory loads .ignore and .gitignore patterns specifically for a directory path.
// It respects the useGitignore and useIgnoreFile flags and adds the exclusionFolder pattern if provided.
// This function should ONLY be called for paths confirmed to be directories.
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

// printSingleFileContent reads and prints the content of a single file.
// If reading fails, it prints a warning to stderr and returns nil error to allow continuation.
func printSingleFileContent(filePath string) error {
	fmt.Printf("File: %s\n", filePath)
	fileData, readErr := os.ReadFile(filePath)

	if readErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to read file %s: %v\n", filePath, readErr)
		fmt.Println("----------------------------------------")
		return nil
	}

	fmt.Println(string(fileData))
	fmt.Printf("End of file: %s\n", filePath)
	fmt.Println("----------------------------------------")
	return nil
}

// deduplicatePatterns removes duplicate patterns from a slice while preserving order.
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
