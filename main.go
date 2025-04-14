package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/temirov/content/commands"
	"github.com/temirov/content/config"
	"github.com/temirov/content/output"
	"github.com/temirov/content/types"
	"github.com/temirov/content/utils"
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
	for _, argumentValue := range os.Args[1:] {
		if argumentValue == flagVersion {
			fmt.Println("content version:", utils.GetApplicationVersion())
			os.Exit(0)
		}
	}

	// parseArgsOrExit will exit if basic parsing fails
	commandName, inputArguments, exclusionFolder, useGitignore, useIgnoreFile, outputFormat := parseArgsOrExit()

	// runContentTool handles processing and prints warnings for non-fatal errors.
	// It returns non-nil error only for fatal issues (validation, rendering).
	executionError := runContentTool(commandName, inputArguments, exclusionFolder, useGitignore, useIgnoreFile, outputFormat)
	if executionError != nil {
		// Log and exit non-zero only for fatal errors returned by runContentTool.
		log.Fatalf("Error: %v", executionError)
	}
	// Exit 0 if runContentTool completed, even if warnings were printed.
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
	fmt.Printf("  For '%s' and '%s': One or more file or directory paths.\n", types.CommandTree, types.CommandContent)
	fmt.Printf("  For '%s': Exactly one fully qualified function name or unique suffix.\n", types.CommandCallChain)
	fmt.Printf("  If no arguments provided for '%s'/'%s', defaults to current directory '%s'.\n", types.CommandTree, types.CommandContent, defaultPath)
	fmt.Printf("\nFlags:\n")
	fmt.Printf("  %s, %s <folder> : Exclude folder name during directory traversal.\n", flagExcludeShort, flagExcludeLong)
	fmt.Printf("  %s           : Disable loading of .gitignore files.\n", flagNoGitignore)
	fmt.Printf("  %s             : Disable loading of .ignore files.\n", flagNoIgnore)
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
			default:
				if currentArgument == flagVersion {
					os.Exit(0)
				}
				fmt.Printf("Error: Unknown flag or misplaced argument: %s\n", currentArgument)
				printUsage()
			}
		} else {
			if parsingFlags {
				fmt.Printf("Error: Positional argument '%s' found after flags.\n", currentArgument)
				printUsage()
			}
			inputArguments = append(inputArguments, currentArgument)
			argumentIndex++
		}
	}

	if (commandName == types.CommandTree || commandName == types.CommandContent) && len(inputArguments) == 0 {
		inputArguments = []string{defaultPath}
	}

	if commandName == types.CommandCallChain && len(inputArguments) != 1 {
		fmt.Printf("Error: The '%s' command requires exactly one function name argument.\n", types.CommandCallChain)
		printUsage()
	}

	return commandName, inputArguments, exclusionFolder, useGitignore, useIgnoreFile, outputFormat
}

// runContentTool orchestrates the main logic: validation, data collection, and rendering.
// It returns a non-nil error only for fatal issues that should cause a non-zero exit code.
// Non-fatal warnings (e.g., read errors) are printed to stderr directly.
func runContentTool(commandName string, inputArguments []string, exclusionFolder string, useGitignore bool, useIgnoreFile bool, outputFormat string) error {
	if commandName == types.CommandCallChain {
		targetFunction := inputArguments[0]
		callChainData, errorCallChain := commands.GetCallChainData(targetFunction)
		if errorCallChain != nil {
			// Consider call chain analysis failure as fatal for this command.
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
			// Failure to render is considered fatal.
			return fmt.Errorf("generating %s output for call chain: %w", outputFormat, renderError)
		}
		fmt.Println(outputString)
		return nil // Success
	}

	// Validate paths first - failure here is fatal.
	validatedPaths, validationError := resolveAndValidatePaths(inputArguments)
	if validationError != nil {
		return validationError
	}

	var collectedResults []interface{}
	// Removed firstProcessingWarning tracking; warnings printed directly.

	for _, pathInfo := range validatedPaths {
		// processingError within the loop is for logging, not returning.
		var processingError error

		if pathInfo.IsDir {
			ignorePatterns, loadErr := config.LoadCombinedIgnorePatterns(pathInfo.AbsolutePath, exclusionFolder, useGitignore, useIgnoreFile)
			if loadErr != nil {
				// Warn about ignore loading error but continue with other paths.
				fmt.Fprintf(os.Stderr, "Warning: Skipping directory %s due to error loading ignore patterns: %v\n", pathInfo.AbsolutePath, loadErr)
				continue // Skip this directory
			}

			switch commandName {
			case types.CommandTree:
				treeNodes, treeErr := commands.GetTreeData(pathInfo.AbsolutePath, ignorePatterns)
				if treeErr != nil {
					// Warn about tree building error but continue.
					processingError = treeErr
				} else if len(treeNodes) > 0 {
					collectedResults = append(collectedResults, treeNodes[0])
				}
			case types.CommandContent:
				fileOutputs, contentErr := commands.GetContentData(pathInfo.AbsolutePath, ignorePatterns)
				if contentErr != nil {
					// Warn about content gathering error but continue.
					processingError = contentErr
				}
				for i := range fileOutputs {
					collectedResults = append(collectedResults, &fileOutputs[i])
				}
			default:
				// This is an internal logic error, should be fatal.
				return fmt.Errorf("internal error: unhandled command '%s' for directory", commandName)
			}
		} else { // Handle explicitly listed files
			switch commandName {
			case types.CommandTree:
				fileNode := &types.TreeOutputNode{
					Path: pathInfo.AbsolutePath,
					Name: filepath.Base(pathInfo.AbsolutePath),
					Type: types.NodeTypeFile,
				}
				collectedResults = append(collectedResults, fileNode)
			case types.CommandContent:
				// getSingleFileContent prints its own warning on error.
				fileOutput, _ := getSingleFileContent(pathInfo.AbsolutePath)
				if fileOutput != nil {
					collectedResults = append(collectedResults, fileOutput)
				}
			default:
				// Internal logic error, should be fatal.
				return fmt.Errorf("internal error: unhandled command '%s' for file", commandName)
			}
		}

		// Log the non-fatal processing error for this specific path if it occurred.
		if processingError != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error processing path %s: %v\n", pathInfo.AbsolutePath, processingError)
		}
	}

	// Render the final results. Rendering failure is considered fatal.
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
	default:
		// Internal logic error, fatal.
		renderError = fmt.Errorf("internal error: unhandled output format '%s'", outputFormat)
	}

	if renderError != nil {
		return fmt.Errorf("error generating %s output: %w", outputFormat, renderError)
	}

	// If we reached here, the overall operation succeeded, even if warnings were printed.
	return nil
}

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
		return nil, fmt.Errorf("error: no valid paths found to process")
	}

	return validatedPaths, nil
}

func getSingleFileContent(filePath string) (*types.FileOutput, error) {
	fileData, readErr := os.ReadFile(filePath)
	if readErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to read file %s: %v\n", filePath, readErr)
		// Return the error so the caller knows *why* it failed, even if it handles it as a warning.
		return nil, readErr
	}

	absolutePath, _ := filepath.Abs(filePath)
	return &types.FileOutput{
		Path:    absolutePath,
		Type:    types.NodeTypeFile,
		Content: string(fileData),
	}, nil
}
