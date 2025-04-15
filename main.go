package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/temirov/ctx/commands"
	"github.com/temirov/ctx/config"
	"github.com/temirov/ctx/output"
	"github.com/temirov/ctx/types"
	"github.com/temirov/ctx/utils"
)

const (
	flagVersion      = "--version"
	flagExcludeShort = "-e"
	flagExcludeLong  = "--e"
	flagNoGitignore  = "--no-gitignore"
	flagNoIgnore     = "--no-ignore"
	flagFormat       = "--format"

	defaultPath = "."
)

func main() {
	// Quick check for version flag before full parsing
	for _, argumentValue := range os.Args[1:] {
		if argumentValue == flagVersion {
			fmt.Println("ctx version:", utils.GetApplicationVersion())
			os.Exit(0)
		}
	}

	commandName, inputArguments, exclusionFolder, useGitignore, useIgnoreFile, outputFormat := parseArgsOrExit()

	// Renamed runContentTool to runTool for clarity
	executionError := runTool(commandName, inputArguments, exclusionFolder, useGitignore, useIgnoreFile, outputFormat)
	if executionError != nil {
		log.Fatalf("Error: %v", executionError)
	}
}

func printUsage() {
	appName := filepath.Base(os.Args[0])
	fmt.Printf("Usage:\n")
	fmt.Printf("  %s <%s|t|%s|c|%s|cc> [arguments...] [flags] [%s <%s|%s>] [%s]\n",
		appName,
		types.CommandTree, types.CommandContent, types.CommandCallChain,
		flagFormat, types.FormatRaw, types.FormatJSON,
		flagVersion)
	fmt.Printf("\nArguments:\n")
	fmt.Printf("  For '%s' and '%s': One or more file or directory paths (defaults to '%s').\n", types.CommandTree, types.CommandContent, defaultPath)
	fmt.Printf("  For '%s': Exactly one fully qualified function name or unique suffix.\n", types.CommandCallChain)
	fmt.Printf("\nFlags:\n")
	fmt.Printf("  %s, %s <folder> : Exclude folder name during directory traversal (for tree/content).\n", flagExcludeShort, flagExcludeLong)
	fmt.Printf("  %s           : Disable loading of .gitignore files (for tree/content).\n", flagNoGitignore)
	fmt.Printf("  %s             : Disable loading of .ignore files (for tree/content).\n", flagNoIgnore)
	fmt.Printf("  %s <%s|%s>   : Set output format (default: %s).\n", flagFormat, types.FormatRaw, types.FormatJSON, types.FormatRaw)
	fmt.Printf("  %s              : Display application version and exit.\n", flagVersion)
	os.Exit(1)
}

func parseArgsOrExit() (commandName string, inputArguments []string, exclusionFolder string, useGitignore bool, useIgnoreFile bool, outputFormat string) {
	if len(os.Args) < 2 {
		printUsage()
	}

	useGitignore = true
	useIgnoreFile = true
	outputFormat = types.FormatRaw

	rawCommand := os.Args[1]
	switch rawCommand {
	case types.CommandTree, "t":
		commandName = types.CommandTree
	case types.CommandContent, "c":
		commandName = types.CommandContent
	case types.CommandCallChain, "cc":
		commandName = types.CommandCallChain
	case flagVersion: // Allow --version as the first argument
		fmt.Println("ctx version:", utils.GetApplicationVersion())
		os.Exit(0)
	default:
		fmt.Printf("Error: Invalid command '%s'\n", rawCommand)
		printUsage()
	}

	arguments := os.Args[2:]
	argumentIndex := 0
	parsingFlags := false

	for argumentIndex < len(arguments) {
		currentArgument := arguments[argumentIndex]
		isFlag := strings.HasPrefix(currentArgument, "-")

		if isFlag {
			parsingFlags = true
			switch currentArgument {
			case flagExcludeShort, flagExcludeLong:
				if argumentIndex+1 >= len(arguments) {
					fmt.Printf("Error: Missing exclusion folder value after %s\n", currentArgument)
					printUsage()
				}
				exclusionFolder = arguments[argumentIndex+1]
				argumentIndex += 2
			case flagNoGitignore:
				useGitignore = false
				argumentIndex++
			case flagNoIgnore:
				useIgnoreFile = false
				argumentIndex++
			case flagFormat:
				if argumentIndex+1 >= len(arguments) {
					fmt.Printf("Error: Missing format value after %s\n", currentArgument)
					printUsage()
				}
				formatValue := strings.ToLower(arguments[argumentIndex+1])
				if formatValue != types.FormatRaw && formatValue != types.FormatJSON {
					fmt.Printf("Error: Invalid format value '%s'. Must be '%s' or '%s'.\n", formatValue, types.FormatRaw, types.FormatJSON)
					printUsage()
				}
				outputFormat = formatValue
				argumentIndex += 2
			case flagVersion: // Handled at the start, but catch here too
				fmt.Println("ctx version:", utils.GetApplicationVersion())
				os.Exit(0)
			default:
				fmt.Printf("Error: Unknown flag or misplaced argument: %s\n", currentArgument)
				printUsage()
			}
		} else {
			// Allow positional argument after flags ONLY for the callchain command's function name
			if parsingFlags && commandName != types.CommandCallChain {
				fmt.Printf("Error: Positional argument '%s' found after flags. Arguments must come before flags for '%s' and '%s' commands.\n", currentArgument, types.CommandTree, types.CommandContent)
				printUsage()
			}
			inputArguments = append(inputArguments, currentArgument)
			argumentIndex++
		}
	}

	// Apply default path if needed for tree/content
	if (commandName == types.CommandTree || commandName == types.CommandContent) && len(inputArguments) == 0 {
		inputArguments = []string{defaultPath}
	}

	// Validate argument count for callchain
	if commandName == types.CommandCallChain && len(inputArguments) != 1 {
		fmt.Printf("Error: The '%s' command requires exactly one function name argument.\n", types.CommandCallChain)
		printUsage()
	}

	// Warn if tree/content specific flags are used with callchain
	if commandName == types.CommandCallChain && (exclusionFolder != "" || !useGitignore || !useIgnoreFile) {
		fmt.Fprintf(os.Stderr, "Warning: Flags -e/--e, --no-gitignore, --no-ignore are ignored for the '%s' command.\n", commandName)
	}

	return commandName, inputArguments, exclusionFolder, useGitignore, useIgnoreFile, outputFormat
}

// runTool orchestrates the main logic based on the parsed command.
func runTool(commandName string, inputArguments []string, exclusionFolder string, useGitignore bool, useIgnoreFile bool, outputFormat string) error {
	switch commandName {
	case types.CommandCallChain:
		return runCallChainCommand(inputArguments[0], outputFormat)
	case types.CommandTree, types.CommandContent:
		return runTreeOrContentCommand(commandName, inputArguments, exclusionFolder, useGitignore, useIgnoreFile, outputFormat)
	default:
		// This should be unreachable due to parsing validation
		return fmt.Errorf("internal error: unhandled command '%s'", commandName)
	}
}

// runCallChainCommand handles the logic for the callchain command.
func runCallChainCommand(targetFunction string, outputFormat string) error {
	callChainData, errorCallChain := commands.GetCallChainData(targetFunction)
	if errorCallChain != nil {
		// Call chain analysis failures are fatal for this command
		return fmt.Errorf("analyzing call chain for '%s': %w", targetFunction, errorCallChain)
	}

	var outputString string
	var renderError error
	if outputFormat == types.FormatJSON {
		outputString, renderError = output.RenderCallChainJSON(callChainData)
	} else {
		outputString = output.RenderCallChainRaw(callChainData)
	}

	if renderError != nil {
		// Rendering failures are fatal
		return fmt.Errorf("generating %s output for call chain: %w", outputFormat, renderError)
	}
	fmt.Println(outputString)
	return nil // Success
}

// runTreeOrContentCommand handles the logic for tree and content commands.
func runTreeOrContentCommand(commandName string, inputArguments []string, exclusionFolder string, useGitignore bool, useIgnoreFile bool, outputFormat string) error {
	// Validate paths first - validation failure is fatal
	validatedPaths, validationError := resolveAndValidatePaths(inputArguments)
	if validationError != nil {
		return validationError
	}

	var collectedResults []interface{} // Holds *FileOutput or *TreeOutputNode

	for _, pathInfo := range validatedPaths {
		var processingError error // Tracks non-fatal error for this specific path

		if pathInfo.IsDir {
			// Load ignore patterns specific to this directory
			ignorePatterns, loadErr := config.LoadCombinedIgnorePatterns(pathInfo.AbsolutePath, exclusionFolder, useGitignore, useIgnoreFile)
			if loadErr != nil {
				// Warn about ignore loading error but continue with other paths
				fmt.Fprintf(os.Stderr, "Warning: Skipping directory %s due to error loading ignore patterns: %v\n", pathInfo.AbsolutePath, loadErr)
				continue // Skip this directory
			}

			// Execute the appropriate command logic for the directory
			switch commandName {
			case types.CommandTree:
				treeNodes, treeErr := commands.GetTreeData(pathInfo.AbsolutePath, ignorePatterns)
				if treeErr != nil {
					processingError = treeErr // Record non-fatal error
				} else if len(treeNodes) > 0 {
					// GetTreeData returns a slice with the root node, add it
					collectedResults = append(collectedResults, treeNodes[0])
				}
			case types.CommandContent:
				fileOutputs, contentErr := commands.GetContentData(pathInfo.AbsolutePath, ignorePatterns)
				if contentErr != nil {
					processingError = contentErr // Record non-fatal error
				}
				// Append pointers to the collected results
				for i := range fileOutputs {
					collectedResults = append(collectedResults, &fileOutputs[i])
				}
			}
		} else { // Handle explicitly listed files (never ignored)
			switch commandName {
			case types.CommandTree:
				// Create a simple file node for the tree output
				fileNode := &types.TreeOutputNode{
					Path: pathInfo.AbsolutePath,
					Name: filepath.Base(pathInfo.AbsolutePath),
					Type: types.NodeTypeFile,
				}
				collectedResults = append(collectedResults, fileNode)
			case types.CommandContent:
				// Get content for the single file
				fileOutput, readErr := getSingleFileContent(pathInfo.AbsolutePath)
				if readErr != nil {
					// Warning already printed by getSingleFileContent
					processingError = readErr // Record non-fatal error
				} else if fileOutput != nil {
					collectedResults = append(collectedResults, fileOutput)
				}
			}
		}

		// Log any non-fatal error encountered for this path before proceeding
		if processingError != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error processing path %s: %v\n", pathInfo.AbsolutePath, processingError)
		}
	}

	// Render the final aggregated results. Rendering failure is considered fatal.
	var renderError error
	switch outputFormat {
	case types.FormatJSON:
		jsonOutputString, jsonErr := output.RenderJSON(collectedResults)
		if jsonErr != nil {
			renderError = jsonErr
		} else {
			fmt.Println(jsonOutputString)
		}
	case types.FormatRaw:
		renderError = output.RenderRaw(commandName, collectedResults)
	default: // Should not happen due to arg parsing
		renderError = fmt.Errorf("internal error: unhandled output format '%s'", outputFormat)
	}

	if renderError != nil {
		return fmt.Errorf("error generating %s output: %w", outputFormat, renderError)
	}

	// If we reached here, the overall operation succeeded, even if warnings were printed.
	return nil
}

// resolveAndValidatePaths checks if paths exist and resolves them to unique absolute paths.
func resolveAndValidatePaths(inputPaths []string) ([]types.ValidatedPath, error) {
	uniquePaths := make(map[string]struct{})
	var validatedPaths []types.ValidatedPath

	if len(inputPaths) == 0 {
		// This should ideally not be reached if defaultPath logic works, but as a safeguard:
		return nil, fmt.Errorf("error: no input paths provided")
	}

	for _, inputPath := range inputPaths {
		absolutePath, absErr := filepath.Abs(inputPath)
		if absErr != nil {
			// Error resolving path itself (e.g., invalid characters)
			return nil, fmt.Errorf("error getting absolute path for '%s': %w", inputPath, absErr)
		}

		cleanPath := filepath.Clean(absolutePath)

		// Skip duplicates
		if _, exists := uniquePaths[cleanPath]; exists {
			continue
		}

		// Check if path exists
		fileInfo, statErr := os.Stat(cleanPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return nil, fmt.Errorf("error: path '%s' (resolved to '%s') does not exist", inputPath, cleanPath)
			}
			// Other stat error (e.g., permission denied to check existence)
			return nil, fmt.Errorf("error stating path '%s' (resolved to '%s'): %w", inputPath, cleanPath, statErr)
		}

		uniquePaths[cleanPath] = struct{}{}
		validatedPaths = append(validatedPaths, types.ValidatedPath{
			AbsolutePath: cleanPath,
			IsDir:        fileInfo.IsDir(),
		})
	}

	// It's possible all input paths were duplicates of the first valid one.
	if len(validatedPaths) == 0 && len(uniquePaths) > 0 {
		// This scenario implies inputPaths contained only duplicates after the first.
		// The logic correctly processed the first unique path, so we find it again.
		// This is a bit inefficient but handles the edge case.
		firstPath := ""
		for p := range uniquePaths {
			firstPath = p
			break
		}
		if firstPath != "" {
			// Re-stat the first unique path to add it back
			fileInfo, _ := os.Stat(firstPath) // Error ignored as it must have succeeded before
			validatedPaths = append(validatedPaths, types.ValidatedPath{
				AbsolutePath: firstPath,
				IsDir:        fileInfo.IsDir(),
			})
		}
	}

	if len(validatedPaths) == 0 {
		// This means no paths were provided, or all provided paths were invalid/duplicates that failed stat.
		return nil, fmt.Errorf("error: no valid paths found to process from the provided arguments")
	}

	return validatedPaths, nil
}

// getSingleFileContent reads content for a single, explicitly provided file path.
// Prints warnings internally on read errors.
func getSingleFileContent(filePath string) (*types.FileOutput, error) {
	fileData, readErr := os.ReadFile(filePath)
	if readErr != nil {
		// Print warning directly to stderr for immediate feedback
		fmt.Fprintf(os.Stderr, "Warning: Failed to read file %s: %v\n", filePath, readErr)
		// Return error so the caller knows it failed, even if it handles it as a warning
		return nil, readErr
	}

	absolutePath, _ := filepath.Abs(filePath) // Error ignored as filePath comes from validated path
	return &types.FileOutput{
		Path:    absolutePath,
		Type:    types.NodeTypeFile,
		Content: string(fileData),
	}, nil
}
