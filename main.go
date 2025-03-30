package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/temirov/content/commands"
	"github.com/temirov/content/config"
	"github.com/temirov/content/types" // Import the new types package
)

// printUsage displays the command-line usage instructions and exits.
func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  content <tree|t|content|c> [path1] [path2] ... [-e|--e exclusion_folder] [--no-gitignore] [--no-ignore] [--format <raw|json>]")
	fmt.Println("\nPaths can be files or directories.")
	fmt.Println("Default format is 'raw'.")
	os.Exit(1)
}

func main() {
	commandName, inputPaths, exclusionFolder, useGitignore, useIgnoreFile, outputFormat := parseArgsOrExit()
	// Use the parsed outputFormat
	err := runContentTool(commandName, inputPaths, exclusionFolder, useGitignore, useIgnoreFile, outputFormat)
	if err != nil {
		// Only log fatal errors that prevent processing/rendering
		log.Fatalf("Error: %v", err)
	}
}

// parseArgsOrExit parses the command-line arguments including the new --format flag.
func parseArgsOrExit() (string, []string, string, bool, bool, string) {
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
	outputFormat := "raw" // Default format

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
			case "--format": // Parse the format flag
				if index+1 >= len(args) {
					fmt.Println("Error: Missing format value after --format")
					printUsage()
				}
				outputFormat = strings.ToLower(args[index+1])
				if outputFormat != "raw" && outputFormat != "json" {
					fmt.Printf("Error: Invalid format value '%s'. Must be 'raw' or 'json'.\n", outputFormat)
					printUsage()
				}
				index += 2
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

	// Return the parsed outputFormat
	return commandName, inputPaths, exclusionFolder, useGitignore, useIgnoreFile, outputFormat
}

// runContentTool orchestrates processing and output generation based on format.
func runContentTool(commandName string, inputPaths []string, exclusionFolder string, useGitignore bool, useIgnoreFile bool, outputFormat string) error {
	validatedPaths, validationErr := resolveAndValidatePaths(inputPaths)
	if validationErr != nil {
		return validationErr // Fatal error during path validation
	}

	// collectedResults holds either *types.FileOutput or *types.TreeOutputNode
	var collectedResults []interface{}
	var firstProcessingWarning error // Track first non-fatal warning/error

	for _, pathInfo := range validatedPaths {
		var processingErr error // Error specific to this path's processing

		if pathInfo.IsDir {
			// Load ignores only for directory processing
			ignorePatterns, loadErr := loadIgnorePatternsForDirectory(pathInfo.AbsolutePath, exclusionFolder, useGitignore, useIgnoreFile)
			if loadErr != nil {
				// Treat failure to load ignores as a warning for this dir
				fmt.Fprintf(os.Stderr, "Warning: Skipping directory %s due to error loading ignore patterns: %v\n", pathInfo.AbsolutePath, loadErr)
				if firstProcessingWarning == nil {
					firstProcessingWarning = loadErr
				}
				continue // Skip this directory
			}

			// Process Directory
			switch commandName {
			case "tree":
				treeNodes, treeErr := commands.GetTreeData(pathInfo.AbsolutePath, ignorePatterns)
				if treeErr != nil {
					processingErr = treeErr // Track error for this path
				} else if len(treeNodes) > 0 {
					collectedResults = append(collectedResults, treeNodes[0]) // Add the root node
				}
			case "content":
				fileOutputs, contentErr := commands.GetContentData(pathInfo.AbsolutePath, ignorePatterns)
				if contentErr != nil {
					processingErr = contentErr // Track error for this path
				} else {
					for i := range fileOutputs {
						collectedResults = append(collectedResults, &fileOutputs[i])
					}
				}
			default:
				processingErr = fmt.Errorf("internal error: unhandled command '%s'", commandName)
			}
		} else {
			// Process File
			switch commandName {
			case "tree":
				fileNode := &types.TreeOutputNode{
					Path: pathInfo.AbsolutePath,
					Name: filepath.Base(pathInfo.AbsolutePath),
					Type: "file",
				}
				collectedResults = append(collectedResults, fileNode)
			case "content":
				// getSingleFileContent handles its own warnings
				fileOutput, _ := getSingleFileContent(pathInfo.AbsolutePath)
				if fileOutput != nil { // Only add if read was successful
					collectedResults = append(collectedResults, fileOutput)
				}
				// Don't assign error here, warning is printed within getSingleFileContent
			default:
				processingErr = fmt.Errorf("internal error: unhandled command '%s'", commandName)
			}
		}

		// Track the first non-fatal processing error
		if processingErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error processing path %s: %v\n", pathInfo.AbsolutePath, processingErr)
			if firstProcessingWarning == nil {
				firstProcessingWarning = processingErr
			}
		}
	}

	// Render the final output based on the chosen format
	var renderErr error
	switch outputFormat {
	case "json":
		renderErr = renderJsonOutput(collectedResults)
	case "raw":
		renderErr = renderRawOutput(commandName, collectedResults)
	default:
		renderErr = fmt.Errorf("internal error: unhandled output format '%s'", outputFormat)
	}

	if renderErr != nil {
		// If rendering fails, that's a fatal error
		return fmt.Errorf("error generating output: %w", renderErr)
	}

	// Return the first processing warning/error encountered, if any
	return firstProcessingWarning
}

// resolveAndValidatePaths converts input paths to absolute paths, checks existence,
// determines if they are files or directories, and removes duplicates.
func resolveAndValidatePaths(inputPaths []string) ([]types.ValidatedPath, error) {
	uniquePaths := make(map[string]struct{})
	var validatedPaths []types.ValidatedPath

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
		validatedPaths = append(validatedPaths, types.ValidatedPath{
			AbsolutePath: cleanPath,
			IsDir:        fileInfo.IsDir(),
		})
	}

	if len(validatedPaths) == 0 {
		// This case should be rare now that "." is default, but keep for safety
		return nil, fmt.Errorf("error: no valid paths found to process")
	}

	return validatedPaths, nil
}

// loadIgnorePatternsForDirectory loads ignore patterns for a specific directory path.
func loadIgnorePatternsForDirectory(absoluteDirectoryPath string, exclusionFolder string, useGitignore bool, useIgnoreFile bool) ([]string, error) {
	var ignorePatterns []string

	if useIgnoreFile {
		ignoreFilePath := filepath.Join(absoluteDirectoryPath, ".ignore")
		loadedIgnorePatterns, loadError := config.LoadContentIgnore(ignoreFilePath)
		if loadError != nil && !os.IsNotExist(loadError) {
			return nil, fmt.Errorf("loading .ignore from %s: %w", absoluteDirectoryPath, loadError)
		}
		ignorePatterns = append(ignorePatterns, loadedIgnorePatterns...)
	}

	if useGitignore {
		gitIgnoreFilePath := filepath.Join(absoluteDirectoryPath, ".gitignore")
		loadedGitignorePatterns, gitignoreErr := config.LoadContentIgnore(gitIgnoreFilePath)
		if gitignoreErr != nil && !os.IsNotExist(gitignoreErr) {
			return nil, fmt.Errorf("loading .gitignore from %s: %w", absoluteDirectoryPath, gitignoreErr)
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

// getSingleFileContent reads content for a single file path.
// Returns nil FileOutput if reading fails (warning printed to stderr).
func getSingleFileContent(filePath string) (*types.FileOutput, error) {
	fileData, readErr := os.ReadFile(filePath)
	if readErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to read file %s: %v\n", filePath, readErr)
		return nil, nil // Indicate skip, not fatal error
	}

	absPath, _ := filepath.Abs(filePath)
	return &types.FileOutput{
		Path:    absPath,
		Type:    "file",
		Content: string(fileData),
	}, nil
}

// renderJsonOutput marshals the collected results and prints to stdout.
func renderJsonOutput(results []interface{}) error {
	if len(results) == 0 {
		fmt.Println("[]") // Print empty JSON array if no results
		return nil
	}
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results to JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

// renderRawOutput iterates through results and prints in the original text format.
func renderRawOutput(commandName string, results []interface{}) error {
	for _, result := range results {
		switch item := result.(type) {
		case *types.FileOutput:
			// Only print file content if the command is 'content'
			if commandName == "content" {
				fmt.Printf("File: %s\n", item.Path)
				fmt.Println(item.Content)
				fmt.Printf("End of file: %s\n", item.Path)
				fmt.Println("----------------------------------------")
			} else {
				// Should not happen for 'tree' command if logic is correct
				fmt.Fprintf(os.Stderr, "Warning: Unexpected FileOutput during raw 'tree' render for path %s\n", item.Path)
			}
		case *types.TreeOutputNode:
			// Only print tree nodes if the command is 'tree'
			if commandName == "tree" {
				if item.Type == "file" {
					fmt.Printf("[File] %s\n", item.Path)
				} else if item.Type == "directory" {
					// Add a blank line before directory trees for better separation
					fmt.Printf("\n--- Directory Tree: %s ---\n", item.Path)
					printRawTreeNode(item, "") // Start recursive print
				}
			} else {
				// Should not happen for 'content' command if logic is correct
				fmt.Fprintf(os.Stderr, "Warning: Unexpected TreeOutputNode during raw 'content' render for path %s\n", item.Path)
			}
		default:
			fmt.Fprintf(os.Stderr, "Warning: Skipping unexpected result type during raw render: %T\n", item)
		}
	}
	return nil
}

// printRawTreeNode recursively prints the tree structure for raw output.
func printRawTreeNode(node *types.TreeOutputNode, prefix string) {
	// Base case: node is nil or not a directory with children
	if node == nil || node.Type != "directory" || len(node.Children) == 0 {
		return
	}

	numChildren := len(node.Children)
	for index, child := range node.Children {
		isLast := index == numChildren-1

		connector := "├── "
		newPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			newPrefix = prefix + "    "
		}

		fmt.Printf("%s%s%s\n", prefix, connector, child.Name)

		// Recurse only if the child is a directory
		if child.Type == "directory" {
			printRawTreeNode(child, newPrefix)
		}
	}
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
