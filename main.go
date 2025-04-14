package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/temirov/content/commands"
	"github.com/temirov/content/config"
	"github.com/temirov/content/types"
)

// GetApplicationVersion returns the application version.
func GetApplicationVersion() string {
	buildInfo, buildInfoAvailable := debug.ReadBuildInfo()
	if buildInfoAvailable && buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
		return buildInfo.Main.Version
	}
	gitDirectory, gitDirectoryError := findGitDirectory(".")
	if gitDirectoryError == nil && gitDirectory != "" {
		gitExactOutput, errorGitExact := exec.Command("git", "describe", "--tags", "--exact-match").Output()
		if errorGitExact == nil && len(gitExactOutput) > 0 {
			return strings.TrimSpace(string(gitExactOutput))
		}
		gitLongOutput, errorGitLong := exec.Command("git", "describe", "--tags", "--long", "--dirty").Output()
		if errorGitLong == nil && len(gitLongOutput) > 0 {
			return strings.TrimSpace(string(gitLongOutput))
		}
	}
	return "unknown"
}

// findGitDirectory searches upward from the starting directory for a .git folder.
func findGitDirectory(startDirectory string) (string, error) {
	absoluteStartDirectory, errorAbsolute := filepath.Abs(startDirectory)
	if errorAbsolute != nil {
		return "", errorAbsolute
	}
	currentDirectory := absoluteStartDirectory
	for {
		gitPath := filepath.Join(currentDirectory, ".git")
		fileInformation, errorStat := os.Stat(gitPath)
		if errorStat == nil && fileInformation.IsDir() {
			return currentDirectory, nil
		}
		parentDirectory := filepath.Dir(currentDirectory)
		if parentDirectory == currentDirectory {
			break
		}
		currentDirectory = parentDirectory
	}
	return "", fmt.Errorf(".git directory not found")
}

func main() {
	for _, argumentValue := range os.Args[1:] {
		if argumentValue == "--version" {
			fmt.Println("content version:", GetApplicationVersion())
			os.Exit(0)
		}
	}
	commandName, inputPaths, exclusionFolder, useGitignore, useIgnoreFile, outputFormat := parseArgsOrExit()
	executionError := runContentTool(commandName, inputPaths, exclusionFolder, useGitignore, useIgnoreFile, outputFormat)
	if executionError != nil {
		log.Fatalf("Error: %v", executionError)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  content <tree|t|content|c|callchain|cc> [arguments...] [flags] [--format <raw|json>] [--version]")
	fmt.Println("Paths can be files or directories (for 'tree' and 'content') or a fully qualified function name (for 'callchain').")
	os.Exit(1)
}

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
	case "callchain", "cc":
		commandName = "callchain"
	default:
		fmt.Printf("Invalid command: %s\n", rawCommand)
		printUsage()
	}
	var inputPaths []string
	exclusionFolder := ""
	useGitignore := true
	useIgnoreFile := true
	outputFormat := "raw"
	arguments := os.Args[2:]
	argumentIndex := 0
	parsingFlags := false
	for argumentIndex < len(arguments) {
		currentArgument := arguments[argumentIndex]
		if strings.HasPrefix(currentArgument, "-") {
			parsingFlags = true
			switch currentArgument {
			case "-e", "--e":
				if argumentIndex+1 >= len(arguments) {
					fmt.Println("Error: Missing exclusion folder value after -e/--e")
					printUsage()
				}
				exclusionFolder = arguments[argumentIndex+1]
				argumentIndex += 2
			case "--no-gitignore":
				useGitignore = false
				argumentIndex++
			case "--no-ignore":
				useIgnoreFile = false
				argumentIndex++
			case "--format":
				if argumentIndex+1 >= len(arguments) {
					fmt.Println("Error: Missing format value after --format")
					printUsage()
				}
				outputFormat = strings.ToLower(arguments[argumentIndex+1])
				if outputFormat != "raw" && outputFormat != "json" {
					fmt.Printf("Error: Invalid format value '%s'. Must be 'raw' or 'json'.\n", outputFormat)
					printUsage()
				}
				argumentIndex += 2
			default:
				fmt.Printf("Error: Unknown flag or misplaced argument: %s\n", currentArgument)
				printUsage()
			}
		} else {
			if parsingFlags {
				fmt.Printf("Error: Positional argument '%s' found after flags.\n", currentArgument)
				printUsage()
			}
			inputPaths = append(inputPaths, currentArgument)
			argumentIndex++
		}
	}
	if len(inputPaths) == 0 {
		inputPaths = []string{"."}
	}
	return commandName, inputPaths, exclusionFolder, useGitignore, useIgnoreFile, outputFormat
}

// runContentTool orchestrates processing and output generation based on format.
func runContentTool(commandName string, inputPaths []string, exclusionFolder string, useGitignore bool, useIgnoreFile bool, outputFormat string) error {
	if commandName == "callchain" {
		if len(inputPaths) != 1 {
			return fmt.Errorf("callchain command requires exactly one function name argument")
		}
		callChainData, errorCallChain := commands.GetCallChainData(inputPaths[0])
		if errorCallChain != nil {
			return errorCallChain
		}
		if outputFormat == "json" {
			jsonOutput, jsonError := commands.RenderCallChainJSON(callChainData)
			if jsonError != nil {
				return fmt.Errorf("error generating JSON output: %w", jsonError)
			}
			fmt.Println(jsonOutput)
		} else {
			fmt.Println(commands.RenderCallChainRaw(callChainData))
		}
		return nil
	}
	validatedPaths, validationError := resolveAndValidatePaths(inputPaths)
	if validationError != nil {
		return validationError
	}
	var collectedResults []interface{}
	var firstProcessingWarning error
	switch commandName {
	case "tree":
		for _, pathInformation := range validatedPaths {
			if pathInformation.IsDir {
				ignorePatterns, errorLoadingIgnores := loadIgnorePatternsForDirectory(pathInformation.AbsolutePath, exclusionFolder, useGitignore, useIgnoreFile)
				if errorLoadingIgnores != nil {
					fmt.Fprintf(os.Stderr, "Warning: Error loading ignore patterns for %s: %v\n", pathInformation.AbsolutePath, errorLoadingIgnores)
					if firstProcessingWarning == nil {
						firstProcessingWarning = errorLoadingIgnores
					}
					continue
				}
				treeNodes, errorGeneratingTree := commands.GetTreeData(pathInformation.AbsolutePath, ignorePatterns)
				if errorGeneratingTree != nil {
					fmt.Fprintf(os.Stderr, "Warning: Error processing path %s: %v\n", pathInformation.AbsolutePath, errorGeneratingTree)
					if firstProcessingWarning == nil {
						firstProcessingWarning = errorGeneratingTree
					}
				} else if len(treeNodes) > 0 {
					collectedResults = append(collectedResults, treeNodes[0])
				}
			} else {
				fileNode := &types.TreeOutputNode{
					Path: pathInformation.AbsolutePath,
					Name: filepath.Base(pathInformation.AbsolutePath),
					Type: "file",
				}
				collectedResults = append(collectedResults, fileNode)
			}
		}
	case "content":
		for _, pathInformation := range validatedPaths {
			if pathInformation.IsDir {
				ignorePatterns, errorLoadingIgnores := loadIgnorePatternsForDirectory(pathInformation.AbsolutePath, exclusionFolder, useGitignore, useIgnoreFile)
				if errorLoadingIgnores != nil {
					fmt.Fprintf(os.Stderr, "Warning: Error loading ignore patterns for %s: %v\n", pathInformation.AbsolutePath, errorLoadingIgnores)
					if firstProcessingWarning == nil {
						firstProcessingWarning = errorLoadingIgnores
					}
					continue
				}
				fileOutputs, errorGeneratingContent := commands.GetContentData(pathInformation.AbsolutePath, ignorePatterns)
				if errorGeneratingContent != nil {
					fmt.Fprintf(os.Stderr, "Warning: Error processing path %s: %v\n", pathInformation.AbsolutePath, errorGeneratingContent)
					if firstProcessingWarning == nil {
						firstProcessingWarning = errorGeneratingContent
					}
				} else {
					for outputIndex := range fileOutputs {
						collectedResults = append(collectedResults, &fileOutputs[outputIndex])
					}
				}
			} else {
				fileOutput, _ := getSingleFileContent(pathInformation.AbsolutePath)
				if fileOutput != nil {
					collectedResults = append(collectedResults, fileOutput)
				}
			}
		}
	default:
		return fmt.Errorf("internal error: unhandled command '%s'", commandName)
	}
	var renderingError error
	switch outputFormat {
	case "json":
		renderingError = renderJsonOutput(collectedResults)
	case "raw":
		renderingError = renderRawOutput(commandName, collectedResults)
	default:
		renderingError = fmt.Errorf("internal error: unhandled output format '%s'", outputFormat)
	}
	if renderingError != nil {
		return fmt.Errorf("error generating output: %w", renderingError)
	}
	return firstProcessingWarning
}

// resolveAndValidatePaths converts input paths to absolute paths, checks existence,
// determines if they are files or directories, and removes duplicates.
func resolveAndValidatePaths(inputPaths []string) ([]types.ValidatedPath, error) {
	uniquePaths := make(map[string]struct{})
	var validatedPaths []types.ValidatedPath
	for _, inputPath := range inputPaths {
		absolutePath, errorGettingAbsolute := filepath.Abs(inputPath)
		if errorGettingAbsolute != nil {
			return nil, fmt.Errorf("error getting absolute path for '%s': %w", inputPath, errorGettingAbsolute)
		}
		cleanPath := filepath.Clean(absolutePath)
		if _, exists := uniquePaths[cleanPath]; exists {
			continue
		}
		fileInformation, errorStat := os.Stat(cleanPath)
		if errorStat != nil {
			if os.IsNotExist(errorStat) {
				return nil, fmt.Errorf("error: path '%s' (resolved to '%s') does not exist", inputPath, cleanPath)
			}
			return nil, fmt.Errorf("error stating path '%s' (resolved to '%s'): %w", inputPath, cleanPath, errorStat)
		}
		uniquePaths[cleanPath] = struct{}{}
		validatedPaths = append(validatedPaths, types.ValidatedPath{
			AbsolutePath: cleanPath,
			IsDir:        fileInformation.IsDir(),
		})
	}
	if len(validatedPaths) == 0 {
		return nil, fmt.Errorf("error: no valid paths found to process")
	}
	return validatedPaths, nil
}

func loadIgnorePatternsForDirectory(directoryPath string, exclusionFolder string, useGitignore bool, useIgnoreFile bool) ([]string, error) {
	var ignorePatterns []string
	if useIgnoreFile {
		ignoreFilePath := filepath.Join(directoryPath, ".ignore")
		loadedIgnorePatterns, errorLoadingIgnore := config.LoadContentIgnore(ignoreFilePath)
		if errorLoadingIgnore != nil && !os.IsNotExist(errorLoadingIgnore) {
			return nil, fmt.Errorf("loading .ignore from %s: %w", directoryPath, errorLoadingIgnore)
		}
		ignorePatterns = append(ignorePatterns, loadedIgnorePatterns...)
	}
	if useGitignore {
		gitIgnoreFilePath := filepath.Join(directoryPath, ".gitignore")
		loadedGitignorePatterns, errorLoadingGitignore := config.LoadContentIgnore(gitIgnoreFilePath)
		if errorLoadingGitignore != nil && !os.IsNotExist(errorLoadingGitignore) {
			return nil, fmt.Errorf("loading .gitignore from %s: %w", directoryPath, errorLoadingGitignore)
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
	fileData, errorReadingFile := os.ReadFile(filePath)
	if errorReadingFile != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to read file %s: %v\n", filePath, errorReadingFile)
		return nil, nil
	}
	absolutePath, _ := filepath.Abs(filePath)
	return &types.FileOutput{
		Path:    absolutePath,
		Type:    "file",
		Content: string(fileData),
	}, nil
}

// renderJsonOutput marshals the collected results and prints to stdout.
func renderJsonOutput(results []interface{}) error {
	if len(results) == 0 {
		fmt.Println("[]")
		return nil
	}
	jsonData, errorMarshalingJson := json.MarshalIndent(results, "", "  ")
	if errorMarshalingJson != nil {
		return fmt.Errorf("failed to marshal results to JSON: %w", errorMarshalingJson)
	}
	fmt.Println(string(jsonData))
	return nil
}

// renderRawOutput iterates through results and prints in the original text format.
func renderRawOutput(commandName string, results []interface{}) error {
	for _, result := range results {
		switch resultTyped := result.(type) {
		case *types.FileOutput:
			if commandName == "content" {
				fmt.Printf("File: %s\n", resultTyped.Path)
				fmt.Println(resultTyped.Content)
				fmt.Printf("End of file: %s\n", resultTyped.Path)
				fmt.Println("----------------------------------------")
			} else {
				fmt.Fprintf(os.Stderr, "Warning: Unexpected FileOutput during raw 'tree' render for path %s\n", resultTyped.Path)
			}
		case *types.TreeOutputNode:
			if commandName == "tree" {
				if resultTyped.Type == "file" {
					fmt.Printf("[File] %s\n", resultTyped.Path)
				} else if resultTyped.Type == "directory" {
					fmt.Printf("\n--- Directory Tree: %s ---\n", resultTyped.Path)
					printRawTreeNode(resultTyped, "")
				}
			} else {
				fmt.Fprintf(os.Stderr, "Warning: Unexpected TreeOutputNode during raw 'content' render for path %s\n", resultTyped.Path)
			}
		default:
			fmt.Fprintf(os.Stderr, "Warning: Skipping unexpected result type during raw render: %T\n", resultTyped)
		}
	}
	return nil
}

func printRawTreeNode(treeNode *types.TreeOutputNode, prefix string) {
	if treeNode == nil || treeNode.Type != "directory" || len(treeNode.Children) == 0 {
		return
	}
	numberOfChildren := len(treeNode.Children)
	for index, child := range treeNode.Children {
		isLastChild := index == numberOfChildren-1
		connector := "├── "
		newPrefix := prefix + "│   "
		if isLastChild {
			connector = "└── "
			newPrefix = prefix + "    "
		}
		fmt.Printf("%s%s%s\n", prefix, connector, child.Name)
		if child.Type == "directory" {
			printRawTreeNode(child, newPrefix)
		}
	}
}

// deduplicatePatterns removes duplicate patterns from a slice while preserving order.
func deduplicatePatterns(patterns []string) []string {
	patternSet := make(map[string]struct{})
	var result []string
	for _, pattern := range patterns {
		if _, exists := patternSet[pattern]; !exists {
			patternSet[pattern] = struct{}{}
			result = append(result, pattern)
		}
	}
	return result
}
