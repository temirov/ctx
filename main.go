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
	fmt.Println("  content <tree|content> [root_directory] [-e exclusion_folder]")
	os.Exit(1)
}

func main() {
	// Manual argument parsing:
	// os.Args[0] is the program name.
	if len(os.Args) < 2 {
		printUsage()
	}

	// The first argument is the command.
	command := os.Args[1]
	if command != "tree" && command != "content" {
		fmt.Printf("Invalid command: %s\n", command)
		printUsage()
	}

	// Set defaults.
	rootDirectory := "."
	exclusionFolder := ""

	// Parse remaining arguments.
	// Expected syntax: [root_directory] [-e exclusion_folder]
	// The root directory is optional; if the first argument after the command starts with "-",
	// then we assume the root directory is omitted.
	arguments := os.Args[2:]
	index := 0
	for index < len(arguments) {
		currentArgument := arguments[index]
		if currentArgument == "-e" {
			if index+1 >= len(arguments) {
				fmt.Println("Error: Missing exclusion folder value after -e")
				printUsage()
			}
			exclusionFolder = arguments[index+1]
			index += 2
		} else {
			// If rootDirectory is still default, assume this argument is the root directory.
			if rootDirectory == "." {
				rootDirectory = currentArgument
			} else {
				fmt.Println("Error: Too many positional arguments")
				printUsage()
			}
			index++
		}
	}

	// Validate the root directory.
	if !utils.IsDirectory(rootDirectory) {
		fmt.Printf("Error: %s is not a valid directory\n", rootDirectory)
		os.Exit(1)
	}

	// Load exclusion patterns from .contentignore in the current working directory.
	currentDirectory, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current working directory: %v", err)
	}
	ignoreFile := filepath.Join(currentDirectory, ".contentignore")
	ignorePatterns, err := config.LoadContentIgnore(ignoreFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error loading .contentignore: %v", err)
	}

	// If the exclusion folder is provided, mark it specially so that it applies only to directories
	// directly under the working/root directory.
	if trimmedExclusion := strings.TrimSpace(exclusionFolder); trimmedExclusion != "" {
		ignorePatterns = append(ignorePatterns, "EXCL:"+trimmedExclusion)
	}

	// Execute the specified command.
	var executionError error
	switch command {
	case "tree":
		executionError = commands.TreeCommand(rootDirectory, ignorePatterns)
	case "content":
		executionError = commands.ContentCommand(rootDirectory, ignorePatterns)
	}

	if executionError != nil {
		log.Fatalf("Error: %v", executionError)
	}
}
