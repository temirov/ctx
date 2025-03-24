// Package main is the entry point for the content CLI tool.
// This updated version implements the following changes:
//  1. The tool now uses a ".ignore" file instead of ".ignore".
//  2. The tool processes ".gitignore" by default (unless disabled with --no-gitignore).
//  3. Two new flags have been added:
//     --no-gitignore  : Disables loading of .gitignore file.
//     --no-ignore     : Disables loading of .ignore file.
//  4. Both -e and --e flags are supported for folder exclusion.
//  5. Command abbreviations "t" for "tree" and "c" for "content" remain supported.
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

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  content <tree|t|content|c> [root_directory] [-e|--e exclusion_folder] [--no-gitignore] [--no-ignore]")
	os.Exit(1)
}

func main() {
	// Parse command-line arguments.
	// useGitignore controls loading of .gitignore (default: true).
	// useIgnoreFile controls loading of .ignore (default: true).
	commandName, rootDirectory, exclusionFolder, useGitignore, useIgnoreFile := parseArgsOrExit()
	err := runContentTool(commandName, rootDirectory, exclusionFolder, useGitignore, useIgnoreFile)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// parseArgsOrExit parses the command-line arguments and returns:
// - the command (normalized to "tree" or "content")
// - the root directory (defaults to ".")
// - the exclusion folder (if provided)
// - a boolean flag indicating whether to load .gitignore (default true)
// - a boolean flag indicating whether to load .ignore (default true)
func parseArgsOrExit() (string, string, string, bool, bool) {
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

	rootDirectory := "."
	exclusionFolder := ""
	// Defaults: load both .gitignore and .ignore files.
	useGitignore := true
	useIgnoreFile := true

	args := os.Args[2:]
	index := 0
	for index < len(args) {
		currentArg := args[index]
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
			// Positional argument for root directory if not already set.
			if rootDirectory == "." {
				rootDirectory = currentArg
			} else {
				fmt.Println("Error: Too many positional arguments")
				printUsage()
			}
			index++
		}
	}
	return commandName, rootDirectory, exclusionFolder, useGitignore, useIgnoreFile
}

// runContentTool validates the root directory, loads ignore patterns,
// and dispatches to the appropriate command.
// It loads ".ignore" and/or ".gitignore"
// based on the provided flags.
func runContentTool(commandName, rootDirectory, exclusionFolder string, useGitignore, useIgnoreFile bool) error {
	if !utils.IsDirectory(rootDirectory) {
		return fmt.Errorf("error: %s is not a valid directory", rootDirectory)
	}

	absoluteRootDirectory, absoluteErr := filepath.Abs(rootDirectory)
	if absoluteErr != nil {
		return absoluteErr
	}

	var ignorePatterns []string

	// Load .ignore file if enabled.
	if useIgnoreFile {
		ignoreFilePath := filepath.Join(absoluteRootDirectory, ".ignore")
		loadedIgnorePatterns, loadError := config.LoadContentIgnore(ignoreFilePath)
		if loadError != nil && !os.IsNotExist(loadError) {
			return fmt.Errorf("error loading .ignore: %w", loadError)
		}
		ignorePatterns = append(ignorePatterns, loadedIgnorePatterns...)
	}

	// Load .gitignore file if enabled.
	if useGitignore {
		gitIgnoreFilePath := filepath.Join(absoluteRootDirectory, ".gitignore")
		loadedGitignorePatterns, gitignoreErr := config.LoadContentIgnore(gitIgnoreFilePath)
		if gitignoreErr != nil && !os.IsNotExist(gitignoreErr) {
			return fmt.Errorf("error loading .gitignore: %w", gitignoreErr)
		}
		ignorePatterns = append(ignorePatterns, loadedGitignorePatterns...)
	}

	// Deduplicate patterns.
	ignorePatterns = deduplicatePatterns(ignorePatterns)

	// Append the additional exclusion flag as a special pattern.
	trimmedExclusion := strings.TrimSpace(exclusionFolder)
	if trimmedExclusion != "" {
		ignorePatterns = append(ignorePatterns, "EXCL:"+trimmedExclusion)
	}

	switch commandName {
	case "tree":
		return commands.TreeCommand(rootDirectory, ignorePatterns)
	case "content":
		return commands.ContentCommand(rootDirectory, ignorePatterns)
	}
	return nil
}

// deduplicatePatterns removes duplicate patterns from a slice.
func deduplicatePatterns(patterns []string) []string {
	patternSet := make(map[string]struct{})
	for _, pattern := range patterns {
		patternSet[pattern] = struct{}{}
	}
	uniquePatterns := make([]string, 0, len(patternSet))
	for pattern := range patternSet {
		uniquePatterns = append(uniquePatterns, pattern)
	}
	return uniquePatterns
}
