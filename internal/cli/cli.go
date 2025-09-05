// Package cli provides the command line interface.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/temirov/ctx/internal/commands"
	"github.com/temirov/ctx/internal/config"
	"github.com/temirov/ctx/internal/docs"
	"github.com/temirov/ctx/internal/output"
	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
)

const (
	exclusionFlagName          = "e"
	noGitignoreFlagName        = "no-gitignore"
	noIgnoreFlagName           = "no-ignore"
	includeGitFlagName         = "git"
	formatFlagName             = "format"
	documentationFlagName      = "doc"
	versionFlagName            = "version"
	versionTemplate            = "ctx version: %s\n"
	documentationIgnoredNotice = "--doc ignored for tree"
	defaultPath                = "."
	rootUse                    = "ctx"
	rootShortDescription       = "ctx command line interface"
	rootLongDescription        = `ctx inspects project structure and source code.
It renders directory trees, shows file content, and analyzes call chains.
Use --format to select raw, json, or xml output, --doc to include documentation, and --version to print the application version.`
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
Use --format to select raw, json, or xml output. The --doc flag is accepted for consistency but ignored.`
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
	documentationFlagDescription    = "include documentation; tree command does not support documentation"
	invalidFormatMessage            = "Invalid format value '%s'"
	warningSkipPathFormat           = "Warning: skipping %s: %v\n"
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
	var documentationEnabled bool

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
			if documentationEnabled {
				fmt.Fprintln(os.Stderr, documentationIgnoredNotice)
				documentationEnabled = false
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
				documentationEnabled,
			)
		},
	}

	addPathFlags(treeCommand, &pathConfiguration)
	treeCommand.Flags().StringVar(&outputFormat, formatFlagName, types.FormatJSON, formatFlagDescription)
	treeCommand.Flags().BoolVar(&documentationEnabled, documentationFlagName, false, documentationFlagDescription)
	return treeCommand
}

// createContentCommand returns the content subcommand.
func createContentCommand() *cobra.Command {
	var pathConfiguration pathOptions
	var outputFormat string = types.FormatJSON
	var documentationEnabled bool

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
			)
		},
	}

	addPathFlags(contentCommand, &pathConfiguration)
	contentCommand.Flags().StringVar(&outputFormat, formatFlagName, types.FormatJSON, formatFlagDescription)
	contentCommand.Flags().BoolVar(&documentationEnabled, documentationFlagName, false, documentationFlagDescription)
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
) error {
	workingDirectory, _ := os.Getwd()
	var collector *docs.Collector
	if documentationEnabled {
		createdCollector, err := docs.NewCollector(workingDirectory)
		if err != nil {
			return err
		}
		collector = createdCollector
	}

	switch commandName {
	case types.CommandCallChain:
		return runCallChain(paths[0], format, callChainDepth, documentationEnabled, collector, workingDirectory)
	case types.CommandTree, types.CommandContent:
		return runTreeOrContentCommand(commandName, paths, exclusionPatterns, useGitignore, useIgnoreFile, includeGit, format, documentationEnabled, collector)
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
	data, err := commands.GetCallChainData(target, callChainDepth, withDocumentation, collector, moduleRoot)
	if err != nil {
		return err
	}
	if format == types.FormatJSON {
		out, err := output.RenderCallChainJSON(data)
		if err != nil {
			return err
		}
		fmt.Println(out)
	} else if format == types.FormatXML {
		out, err := output.RenderCallChainXML(data)
		if err != nil {
			return err
		}
		fmt.Println(out)
	} else {
		fmt.Println(output.RenderCallChainRaw(data))
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
	collector *docs.Collector,
) error {
	validatedPaths, err := resolveAndValidatePaths(paths)
	if err != nil {
		return err
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
				nodes, dataError := commands.GetTreeData(info.AbsolutePath, ignorePatternList)
				if dataError == nil && len(nodes) > 0 {
					collected = append(collected, nodes[0])
				}
			} else {
				files, dataError := commands.GetContentData(info.AbsolutePath, ignorePatternList, binaryContentPatternList)
				if dataError == nil {
					for index := range files {
						collected = append(collected, &files[index])
					}
				}
			}
		} else {
			if commandName == types.CommandTree {
				mimeType := utils.DetectMimeType(info.AbsolutePath)
				nodeType := types.NodeTypeFile
				if utils.IsFileBinary(info.AbsolutePath) {
					nodeType = types.NodeTypeBinary
				}
				collected = append(collected, &types.TreeOutputNode{
					Path:     info.AbsolutePath,
					Name:     filepath.Base(info.AbsolutePath),
					Type:     nodeType,
					MimeType: mimeType,
				})
			} else {
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
				collected = append(collected, &types.FileOutput{
					Path:     info.AbsolutePath,
					Type:     fileType,
					Content:  content,
					MimeType: mimeType,
				})
			}
		}
	}

	var documentationEntries []types.DocumentationEntry
	if withDocumentation && collector != nil {
		for _, item := range collected {
			fileOutput, ok := item.(*types.FileOutput)
			if ok {
				entries, _ := collector.CollectFromFile(fileOutput.Path)
				documentationEntries = append(documentationEntries, entries...)
			}
		}
		sort.Slice(documentationEntries, func(i, j int) bool {
			if documentationEntries[i].Kind != documentationEntries[j].Kind {
				return documentationEntries[i].Kind < documentationEntries[j].Kind
			}
			return documentationEntries[i].Name < documentationEntries[j].Name
		})
	}

	if format == types.FormatJSON {
		out, err := output.RenderJSON(documentationEntries, collected)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	} else if format == types.FormatXML {
		out, err := output.RenderXML(documentationEntries, collected)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	return output.RenderRaw(commandName, documentationEntries, collected)
}

// resolveAndValidatePaths converts input paths to absolute form and validates their existence.
func resolveAndValidatePaths(inputs []string) ([]types.ValidatedPath, error) {
	seen := make(map[string]struct{})
	var result []types.ValidatedPath
	for _, inputPath := range inputs {
		absolutePath, err := filepath.Abs(inputPath)
		if err != nil {
			return nil, fmt.Errorf("abs failed for '%s': %w", inputPath, err)
		}
		cleanPath := filepath.Clean(absolutePath)
		if _, ok := seen[cleanPath]; ok {
			continue
		}
		info, err := os.Stat(cleanPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("path '%s' does not exist", inputPath)
			}
			return nil, fmt.Errorf("stat failed for '%s': %w", inputPath, err)
		}
		seen[cleanPath] = struct{}{}
		result = append(result, types.ValidatedPath{AbsolutePath: cleanPath, IsDir: info.IsDir()})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid paths")
	}
	return result, nil
}
