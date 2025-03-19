// Package main is the entry point for the content CLI tool.
// It parses command-line arguments and dispatches to tree/content commands.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	//nolint:depguard
	"github.com/temirov/content/commands"
	//nolint:depguard
	"github.com/temirov/content/config"
	//nolint:depguard
	"github.com/temirov/content/utils"
)

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  content <tree|content> [root_directory] [-e exclusion_folder]")
	os.Exit(1)
}

func main() {
	commandName, rootDirectory, exclusionFolder := parseArgsOrExit()
	err := runContentTool(commandName, rootDirectory, exclusionFolder)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func parseArgsOrExit() (string, string, string) {
	if len(os.Args) < 2 {
		printUsage()
	}

	commandName := os.Args[1]
	if commandName != "tree" && commandName != "content" {
		fmt.Printf("Invalid command: %s\n", commandName)
		printUsage()
	}

	rootDirectory := "."
	exclusionFolder := ""

	args := os.Args[2:]
	index := 0
	for index < len(args) {
		currentArg := args[index]
		if currentArg == "-e" {
			if index+1 >= len(args) {
				fmt.Println("Error: Missing exclusion folder value after -e")
				printUsage()
			}
			exclusionFolder = args[index+1]
			index += 2
		} else {
			if rootDirectory == "." {
				rootDirectory = currentArg
			} else {
				fmt.Println("Error: Too many positional arguments")
				printUsage()
			}
			index++
		}
	}
	return commandName, rootDirectory, exclusionFolder
}

func runContentTool(commandName, rootDirectory, exclusionFolder string) error {
	if !utils.IsDirectory(rootDirectory) {
		// ST1005: do not capitalize "error"
		return fmt.Errorf("error: %s is not a valid directory", rootDirectory)
	}

	absoluteRootDirectory, absoluteErr := filepath.Abs(rootDirectory)
	if absoluteErr != nil {
		return absoluteErr
	}
	ignoreFilePath := filepath.Join(absoluteRootDirectory, ".contentignore")

	ignorePatterns, loadError := config.LoadContentIgnore(ignoreFilePath)
	if loadError != nil && !os.IsNotExist(loadError) {
		// ST1005: do not capitalize "error"
		return fmt.Errorf("error loading .contentignore: %w", loadError)
	}

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
