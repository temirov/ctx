// Package cli provides the command line interface.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/temirov/ctx/internal/commands"
	"github.com/temirov/ctx/internal/config"
	"github.com/temirov/ctx/internal/docs"
	"github.com/temirov/ctx/internal/output"
	"github.com/temirov/ctx/internal/tokenizer"
	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

const (
	exclusionFlagName     = "e"
	noGitignoreFlagName   = "no-gitignore"
	noIgnoreFlagName      = "no-ignore"
	includeGitFlagName    = "git"
	formatFlagName        = "format"
	summaryFlagName       = "summary"
	tokensFlagName        = "tokens"
	modelFlagName         = "model"
	pythonFlagName        = "python"
	sentencePieceFlagName = "spm-model"
	pythonHelpersFlagName = "py-helpers-dir"
	documentationFlagName = "doc"
	versionFlagName       = "version"
	versionTemplate       = "ctx version: %s\n"
	defaultPath           = "."
	rootUse               = "ctx"
	rootShortDescription  = "ctx command line interface"
	rootLongDescription   = `ctx inspects project structure and source code.
It renders directory trees, shows file content, and analyzes call chains.
Use --format to select raw, json, or xml output. Use --doc to include documentation for supported commands, and --version to print the application version.`
	versionFlagDescription    = "display application version"
	treeUse                   = "tree [paths...]"
	contentUse                = "content [paths...]"
	callchainUse              = "callchain <function>"
	treeAlias                 = "t"
	contentAlias              = "c"
	callchainAlias            = "cc"
	treeShortDescription      = "display directory tree (" + treeAlias + ")"
	contentShortDescription   = "show file contents (" + contentAlias + ")"
	callchainShortDescription = "analyze call chains (" + callchainAlias + ")"

	// treeLongDescription provides detailed help for the tree command.
	treeLongDescription = `List directories and files for one or more paths.
Use --format to select raw, json, or xml output.`
	// treeUsageExample demonstrates tree command usage.
	treeUsageExample = `  # Render the tree in XML format
  ctx tree --format xml ./cmd

  # Exclude vendor directory
  ctx tree -e vendor .`

	// contentLongDescription provides detailed help for the content command.
	contentLongDescription = `Display file content for provided paths.
Use --format to select raw, json, or xml output and --doc to include collected documentation.`
	// contentUsageExample demonstrates content command usage.
	contentUsageExample = `  # Show project files with documentation
  ctx content --doc .

  # Display a file in raw format
  ctx content --format raw main.go`

	// callchainLongDescription provides detailed help for the callchain command.
	callchainLongDescription = `Analyze the call chain of a function.
Use --depth to control traversal depth, --format for output selection, and --doc to include documentation.`
	// callchainUsageExample demonstrates callchain command usage.
	callchainUsageExample = `  # Analyze call chain up to depth two in JSON
  ctx callchain fmt.Println --depth 2

  # Produce XML output including documentation
  ctx callchain mypkg.MyFunc --format xml --doc`

	callChainDepthFlagName          = "depth"
	unsupportedCommandMessage       = "unsupported command"
	defaultCallChainDepth           = 1
	callChainDepthDescription       = "traversal depth"
	exclusionFlagDescription        = "exclude path pattern"
	disableGitignoreFlagDescription = "do not use .gitignore"
	disableIgnoreFlagDescription    = "do not use .ignore"
	includeGitFlagDescription       = "include git directory"
	formatFlagDescription           = "output format"
	summaryFlagDescription          = "include summary of resulting files"
	tokensFlagDescription           = "include token counts"
	modelFlagDescription            = "tokenizer model to use for token counting"
	pythonFlagDescription           = "python executable to use for external tokenizers"
	sentencePieceDescription        = "SentencePiece tokenizer.model path for llama models"
	pythonHelpersDescription        = "directory containing tokenizer helper scripts"
	defaultTokenizerModelName       = "gpt-4o"
	defaultPythonExecutable         = "python3"
	documentationFlagDescription    = "include documentation"
	invalidFormatMessage            = "Invalid format value '%s'"
	warningSkipPathFormat           = "Warning: skipping %s: %v\n"
	warningTokenCountFormat         = "Warning: failed to count tokens for %s: %v\n"
	workingDirectoryErrorFormat     = "unable to determine working directory: %w"

	// errorAbsolutePathFormat reports failure to resolve an absolute path.
	errorAbsolutePathFormat = "abs failed for '%s': %w"
	// errorPathMissingFormat reports a missing path.
	errorPathMissingFormat = "path '%s' does not exist"
	// errorStatFormat reports failure to retrieve file statistics.
	errorStatFormat = "stat failed for '%s': %w"
	// errorNoValidPaths indicates that all paths are invalid.
	errorNoValidPaths = "no valid paths"
)

// isSupportedFormat reports whether the provided format is recognized.
func isSupportedFormat(format string) bool {
	switch format {
	case types.FormatRaw, types.FormatJSON, types.FormatXML:
		return true
	default:
		return false
	}
}

// Execute runs the ctx application.
func Execute() error {
	rootCommand := createRootCommand()
	return rootCommand.Execute()
}

// createRootCommand builds the root Cobra command.
func createRootCommand() *cobra.Command {
	var showVersion bool

	rootCommand := &cobra.Command{
		Use:          rootUse,
		Short:        rootShortDescription,
		Long:         rootLongDescription,
		SilenceUsage: true,
		RunE: func(command *cobra.Command, arguments []string) error {
			return command.Help()
		},
		PersistentPreRun: func(command *cobra.Command, arguments []string) {
			if showVersion {
				fmt.Printf(versionTemplate, utils.GetApplicationVersion())
				os.Exit(0)
			}
		},
	}
	rootCommand.PersistentFlags().BoolVar(&showVersion, versionFlagName, false, versionFlagDescription)
	rootCommand.AddCommand(
		createTreeCommand(),
		createContentCommand(),
		createCallChainCommand(),
	)
	rootCommand.InitDefaultHelpCmd()
	rootCommand.InitDefaultCompletionCmd()
	return rootCommand
}

// pathOptions stores configuration for path-related flags.
type pathOptions struct {
	exclusionPatterns []string
	disableGitignore  bool
	disableIgnoreFile bool
	includeGit        bool
}

type tokenOptions struct {
	enabled            bool
	model              string
	pythonExecutable   string
	sentencePieceModel string
	pythonHelpersDir   string
}

func (options tokenOptions) toConfig(workingDirectory string) tokenizer.Config {
	return tokenizer.Config{
		Model:                  options.model,
		PythonExecutable:       options.pythonExecutable,
		HelpersDir:             options.pythonHelpersDir,
		SentencePieceModelPath: options.sentencePieceModel,
		WorkingDirectory:       workingDirectory,
	}
}

// addPathFlags registers path-related flags on the command.
func addPathFlags(command *cobra.Command, options *pathOptions) {
	command.Flags().StringArrayVarP(&options.exclusionPatterns, exclusionFlagName, exclusionFlagName, nil, exclusionFlagDescription)
	command.Flags().BoolVar(&options.disableGitignore, noGitignoreFlagName, false, disableGitignoreFlagDescription)
	command.Flags().BoolVar(&options.disableIgnoreFile, noIgnoreFlagName, false, disableIgnoreFlagDescription)
	command.Flags().BoolVar(&options.includeGit, includeGitFlagName, false, includeGitFlagDescription)
}

// createTreeCommand returns the tree subcommand.
func createTreeCommand() *cobra.Command {
	var pathConfiguration pathOptions
	var outputFormat string = types.FormatJSON
	var summaryEnabled bool
	var tokenConfiguration tokenOptions
	tokenConfiguration.model = defaultTokenizerModelName
	tokenConfiguration.pythonExecutable = defaultPythonExecutable

	treeCommand := &cobra.Command{
		Use:     treeUse,
		Aliases: []string{treeAlias},
		Short:   treeShortDescription,
		Long:    treeLongDescription,
		Example: treeUsageExample,
		Args:    cobra.ArbitraryArgs,
		RunE: func(command *cobra.Command, arguments []string) error {
			if len(arguments) == 0 {
				arguments = []string{defaultPath}
			}
			outputFormatLower := strings.ToLower(outputFormat)
			if !isSupportedFormat(outputFormatLower) {
				return fmt.Errorf(invalidFormatMessage, outputFormatLower)
			}
			return runTool(
				types.CommandTree,
				arguments,
				pathConfiguration.exclusionPatterns,
				!pathConfiguration.disableGitignore,
				!pathConfiguration.disableIgnoreFile,
				pathConfiguration.includeGit,
				defaultCallChainDepth,
				outputFormatLower,
				false,
				summaryEnabled,
				tokenConfiguration,
			)
		},
	}

	addPathFlags(treeCommand, &pathConfiguration)
	treeCommand.Flags().StringVar(&outputFormat, formatFlagName, types.FormatJSON, formatFlagDescription)
	treeCommand.Flags().BoolVar(&summaryEnabled, summaryFlagName, false, summaryFlagDescription)
	treeCommand.Flags().BoolVar(&tokenConfiguration.enabled, tokensFlagName, false, tokensFlagDescription)
	treeCommand.Flags().StringVar(&tokenConfiguration.model, modelFlagName, defaultTokenizerModelName, modelFlagDescription)
	treeCommand.Flags().StringVar(&tokenConfiguration.pythonExecutable, pythonFlagName, defaultPythonExecutable, pythonFlagDescription)
	treeCommand.Flags().StringVar(&tokenConfiguration.sentencePieceModel, sentencePieceFlagName, "", sentencePieceDescription)
	treeCommand.Flags().StringVar(&tokenConfiguration.pythonHelpersDir, pythonHelpersFlagName, "", pythonHelpersDescription)
	return treeCommand
}

// createContentCommand returns the content subcommand.
func createContentCommand() *cobra.Command {
	var pathConfiguration pathOptions
	var outputFormat string = types.FormatJSON
	var documentationEnabled bool
	var summaryEnabled bool
	var tokenConfiguration tokenOptions
	tokenConfiguration.model = defaultTokenizerModelName
	tokenConfiguration.pythonExecutable = defaultPythonExecutable

	contentCommand := &cobra.Command{
		Use:     contentUse,
		Aliases: []string{contentAlias},
		Short:   contentShortDescription,
		Long:    contentLongDescription,
		Example: contentUsageExample,
		Args:    cobra.ArbitraryArgs,
		RunE: func(command *cobra.Command, arguments []string) error {
			if len(arguments) == 0 {
				arguments = []string{defaultPath}
			}
			outputFormatLower := strings.ToLower(outputFormat)
			if !isSupportedFormat(outputFormatLower) {
				return fmt.Errorf(invalidFormatMessage, outputFormatLower)
			}
			return runTool(
				types.CommandContent,
				arguments,
				pathConfiguration.exclusionPatterns,
				!pathConfiguration.disableGitignore,
				!pathConfiguration.disableIgnoreFile,
				pathConfiguration.includeGit,
				defaultCallChainDepth,
				outputFormatLower,
				documentationEnabled,
				summaryEnabled,
				tokenConfiguration,
			)
		},
	}

	addPathFlags(contentCommand, &pathConfiguration)
	contentCommand.Flags().StringVar(&outputFormat, formatFlagName, types.FormatJSON, formatFlagDescription)
	contentCommand.Flags().BoolVar(&documentationEnabled, documentationFlagName, false, documentationFlagDescription)
	contentCommand.Flags().BoolVar(&summaryEnabled, summaryFlagName, false, summaryFlagDescription)
	contentCommand.Flags().BoolVar(&tokenConfiguration.enabled, tokensFlagName, false, tokensFlagDescription)
	contentCommand.Flags().StringVar(&tokenConfiguration.model, modelFlagName, defaultTokenizerModelName, modelFlagDescription)
	contentCommand.Flags().StringVar(&tokenConfiguration.pythonExecutable, pythonFlagName, defaultPythonExecutable, pythonFlagDescription)
	contentCommand.Flags().StringVar(&tokenConfiguration.sentencePieceModel, sentencePieceFlagName, "", sentencePieceDescription)
	contentCommand.Flags().StringVar(&tokenConfiguration.pythonHelpersDir, pythonHelpersFlagName, "", pythonHelpersDescription)
	return contentCommand
}

// createCallChainCommand returns the callchain subcommand.
func createCallChainCommand() *cobra.Command {
	var outputFormat string = types.FormatJSON
	var documentationEnabled bool
	var callChainDepth int = defaultCallChainDepth

	callChainCommand := &cobra.Command{
		Use:     callchainUse,
		Aliases: []string{callchainAlias},
		Short:   callchainShortDescription,
		Long:    callchainLongDescription,
		Example: callchainUsageExample,
		Args:    cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			outputFormatLower := strings.ToLower(outputFormat)
			if !isSupportedFormat(outputFormatLower) {
				return fmt.Errorf(invalidFormatMessage, outputFormatLower)
			}
			return runTool(
				types.CommandCallChain,
				[]string{arguments[0]},
				nil,
				true,
				true,
				false,
				callChainDepth,
				outputFormatLower,
				documentationEnabled,
				false,
				tokenOptions{},
			)
		},
	}

	callChainCommand.Flags().StringVar(&outputFormat, formatFlagName, types.FormatJSON, formatFlagDescription)
	callChainCommand.Flags().BoolVar(&documentationEnabled, documentationFlagName, false, documentationFlagDescription)
	callChainCommand.Flags().IntVar(&callChainDepth, callChainDepthFlagName, defaultCallChainDepth, callChainDepthDescription)
	return callChainCommand
}

// runTool executes the command with the provided configuration including call chain depth.
func runTool(
	commandName string,
	paths []string,
	exclusionPatterns []string,
	useGitignore bool,
	useIgnoreFile bool,
	includeGit bool,
	callChainDepth int,
	format string,
	documentationEnabled bool,
	summaryEnabled bool,
	tokenConfiguration tokenOptions,
) error {
	workingDirectory, workingDirectoryError := os.Getwd()
	if workingDirectoryError != nil {
		return fmt.Errorf(workingDirectoryErrorFormat, workingDirectoryError)
	}
	var collector *docs.Collector
	if documentationEnabled {
		createdCollector, collectorCreationError := docs.NewCollector(workingDirectory)
		if collectorCreationError != nil {
			return collectorCreationError
		}
		collector = createdCollector
	}

	var tokenCounter tokenizer.Counter
	if tokenConfiguration.enabled {
		createdCounter, counterError := tokenizer.NewCounter(tokenConfiguration.toConfig(workingDirectory))
		if counterError != nil {
			return counterError
		}
		tokenCounter = createdCounter
	}

	switch commandName {
	case types.CommandCallChain:
		return runCallChain(paths[0], format, callChainDepth, documentationEnabled, collector, workingDirectory)
	case types.CommandTree, types.CommandContent:
		return runTreeOrContentCommand(commandName, paths, exclusionPatterns, useGitignore, useIgnoreFile, includeGit, format, documentationEnabled, summaryEnabled, tokenCounter, collector)
	default:
		return fmt.Errorf(unsupportedCommandMessage)
	}
}

// runCallChain processes the callchain command for the specified target and depth.
func runCallChain(
	target string,
	format string,
	callChainDepth int,
	withDocumentation bool,
	collector *docs.Collector,
	moduleRoot string,
) error {
	callChainData, callChainDataError := commands.GetCallChainData(target, callChainDepth, withDocumentation, collector, moduleRoot)
	if callChainDataError != nil {
		return callChainDataError
	}
	if format == types.FormatJSON {
		renderedCallChainJSONOutput, renderCallChainJSONError := output.RenderCallChainJSON(callChainData)
		if renderCallChainJSONError != nil {
			return renderCallChainJSONError
		}
		fmt.Println(renderedCallChainJSONOutput)
	} else if format == types.FormatXML {
		renderedCallChainXMLOutput, renderCallChainXMLError := output.RenderCallChainXML(callChainData)
		if renderCallChainXMLError != nil {
			return renderCallChainXMLError
		}
		fmt.Println(renderedCallChainXMLOutput)
	} else {
		fmt.Println(output.RenderCallChainRaw(callChainData))
	}
	return nil
}

// runTreeOrContentCommand executes tree or content commands for the given paths.
func runTreeOrContentCommand(
	commandName string,
	paths []string,
	exclusionPatterns []string,
	useGitignore bool,
	useIgnoreFile bool,
	includeGit bool,
	format string,
	withDocumentation bool,
	withSummary bool,
	tokenCounter tokenizer.Counter,
	collector *docs.Collector,
) error {
	validatedPaths, pathValidationError := resolveAndValidatePaths(paths)
	if pathValidationError != nil {
		return pathValidationError
	}

	var collected []interface{}
	for _, info := range validatedPaths {
		if info.IsDir {
			ignorePatternList, binaryContentPatternList, loadError := config.LoadRecursiveIgnorePatterns(info.AbsolutePath, exclusionPatterns, useGitignore, useIgnoreFile, includeGit)
			if loadError != nil {
				fmt.Fprintf(os.Stderr, warningSkipPathFormat, info.AbsolutePath, loadError)
				continue
			}
			if commandName == types.CommandTree {
				treeBuilder := commands.TreeBuilder{IgnorePatterns: ignorePatternList, IncludeSummary: withSummary, TokenCounter: tokenCounter}
				nodes, dataError := treeBuilder.GetTreeData(info.AbsolutePath)
				if dataError == nil && len(nodes) > 0 {
					collected = append(collected, nodes[0])
				}
			} else {
				files, dataError := commands.GetContentData(info.AbsolutePath, ignorePatternList, binaryContentPatternList, tokenCounter)
				if dataError == nil {
					for index := range files {
						if withDocumentation && collector != nil {
							entries, _ := collector.CollectFromFile(files[index].Path)
							files[index].Documentation = entries
						}
						collected = append(collected, &files[index])
					}
					contentRoot, contentTreeError := commands.BuildContentTree(info.AbsolutePath, files, withSummary)
					if contentTreeError != nil {
						fmt.Fprintf(os.Stderr, warningSkipPathFormat, info.AbsolutePath, contentTreeError)
					} else if contentRoot != nil {
						collected = append(collected, contentRoot)
					}
				}
			}
		} else {
			if commandName == types.CommandTree {
				fileInfo, statError := os.Stat(info.AbsolutePath)
				if statError != nil {
					fmt.Fprintf(os.Stderr, warningSkipPathFormat, info.AbsolutePath, statError)
					continue
				}
				mimeType := utils.DetectMimeType(info.AbsolutePath)
				nodeType := types.NodeTypeFile
				if utils.IsFileBinary(info.AbsolutePath) {
					nodeType = types.NodeTypeBinary
				}
				node := &types.TreeOutputNode{
					Path:         info.AbsolutePath,
					Name:         filepath.Base(info.AbsolutePath),
					Type:         nodeType,
					Size:         utils.FormatFileSize(fileInfo.Size()),
					SizeBytes:    fileInfo.Size(),
					LastModified: utils.FormatTimestamp(fileInfo.ModTime()),
					MimeType:     mimeType,
				}
				var tokenCount int
				if tokenCounter != nil && nodeType != types.NodeTypeBinary {
					countResult, countErr := tokenizer.CountFile(tokenCounter, info.AbsolutePath)
					if countErr != nil {
						fmt.Fprintf(os.Stderr, warningTokenCountFormat, info.AbsolutePath, countErr)
					} else if countResult.Counted {
						tokenCount = countResult.Tokens
					}
				}
				node.Tokens = tokenCount
				if withSummary {
					node.TotalFiles = 1
					node.TotalSize = node.Size
					node.TotalTokens = tokenCount
				}
				collected = append(collected, node)
			} else {
				fileInfo, statError := os.Stat(info.AbsolutePath)
				if statError != nil {
					fmt.Fprintf(os.Stderr, warningSkipPathFormat, info.AbsolutePath, statError)
					continue
				}
				fileBytes, fileReadError := os.ReadFile(info.AbsolutePath)
				if fileReadError != nil {
					fmt.Fprintf(os.Stderr, commands.WarningFileReadFormat, info.AbsolutePath, fileReadError)
					continue
				}
				mimeType := utils.DetectMimeType(info.AbsolutePath)
				fileType := types.NodeTypeFile
				content := string(fileBytes)
				if utils.IsBinary(fileBytes) {
					fileType = types.NodeTypeBinary
					content = ""
				}
				var tokenCount int
				if tokenCounter != nil && fileType != types.NodeTypeBinary {
					countResult, countErr := tokenizer.CountBytes(tokenCounter, fileBytes)
					if countErr != nil {
						fmt.Fprintf(os.Stderr, warningTokenCountFormat, info.AbsolutePath, countErr)
					} else if countResult.Counted {
						tokenCount = countResult.Tokens
					}
				}
				fileOutput := types.FileOutput{
					Path:         info.AbsolutePath,
					Type:         fileType,
					Content:      content,
					Size:         utils.FormatFileSize(fileInfo.Size()),
					SizeBytes:    fileInfo.Size(),
					LastModified: utils.FormatTimestamp(fileInfo.ModTime()),
					MimeType:     mimeType,
					Tokens:       tokenCount,
				}
				if withDocumentation && collector != nil {
					entries, _ := collector.CollectFromFile(fileOutput.Path)
					fileOutput.Documentation = entries
				}
				collected = append(collected, &fileOutput)
				contentRoot, contentTreeError := commands.BuildContentTree(info.AbsolutePath, []types.FileOutput{fileOutput}, withSummary)
				if contentTreeError != nil {
					fmt.Fprintf(os.Stderr, warningSkipPathFormat, info.AbsolutePath, contentTreeError)
				} else if contentRoot != nil {
					collected = append(collected, contentRoot)
				}
			}
		}
	}

	if format == types.FormatJSON {
		renderedJSONOutput, renderJSONError := output.RenderJSON(collected)
		if renderJSONError != nil {
			return renderJSONError
		}
		fmt.Println(renderedJSONOutput)
		return nil
	} else if format == types.FormatXML {
		renderedXMLOutput, renderXMLError := output.RenderXML(collected)
		if renderXMLError != nil {
			return renderXMLError
		}
		fmt.Println(renderedXMLOutput)
		return nil
	}

	return output.RenderRaw(commandName, collected, withSummary)
}

// resolveAndValidatePaths converts input paths to absolute form and validates their existence.
func resolveAndValidatePaths(inputs []string) ([]types.ValidatedPath, error) {
	seen := make(map[string]struct{})
	var result []types.ValidatedPath
	for _, inputPath := range inputs {
		absolutePath, absolutePathError := filepath.Abs(inputPath)
		if absolutePathError != nil {
			return nil, fmt.Errorf(errorAbsolutePathFormat, inputPath, absolutePathError)
		}
		cleanPath := filepath.Clean(absolutePath)
		if _, ok := seen[cleanPath]; ok {
			continue
		}
		info, fileStatusError := os.Stat(cleanPath)
		if fileStatusError != nil {
			if os.IsNotExist(fileStatusError) {
				return nil, fmt.Errorf(errorPathMissingFormat, inputPath)
			}
			return nil, fmt.Errorf(errorStatFormat, inputPath, fileStatusError)
		}
		seen[cleanPath] = struct{}{}
		result = append(result, types.ValidatedPath{AbsolutePath: cleanPath, IsDir: info.IsDir()})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf(errorNoValidPaths)
	}
	return result, nil
}
