package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/temirov/ctx/commands"
	"github.com/temirov/ctx/config"
	"github.com/temirov/ctx/docs"
	"github.com/temirov/ctx/output"
	"github.com/temirov/ctx/types"
	"github.com/temirov/ctx/utils"
)

const (
	flagVersionName     = "version"
	flagExcludeName     = "e"
	flagNoGitignoreName = "no-gitignore"
	flagNoIgnoreName    = "no-ignore"
	flagFormatName      = "format"
	flagDocName         = "doc"
	defaultPath         = "."
	versionPrefix       = "ctx version:"
	docIgnoredMessage   = "--doc ignored for tree"
)

var (
	showVersionFlag      bool
	exclusionFolder      string
	noGitignoreFlag      bool
	noIgnoreFlag         bool
	outputFormat         string
	documentationEnabled bool
)

// main is the entry point of the ctx CLI.
func main() {
	rootCommand := newRootCommand()
	if executeError := rootCommand.Execute(); executeError != nil {
		log.Fatalf("Error: %v", executeError)
	}
}

// newRootCommand constructs the root Cobra command with all subcommands.
func newRootCommand() *cobra.Command {
	rootCommand := &cobra.Command{
		Use:          "ctx",
		SilenceUsage: true,
		RunE: func(command *cobra.Command, arguments []string) error {
			return command.Help()
		},
	}

	persistentFlags := rootCommand.PersistentFlags()
	persistentFlags.BoolVar(&showVersionFlag, flagVersionName, false, "Show application version")
	persistentFlags.StringVarP(&exclusionFolder, flagExcludeName, flagExcludeName, "", "Folder to exclude")
	persistentFlags.BoolVar(&noGitignoreFlag, flagNoGitignoreName, false, "Disable use of .gitignore files")
	persistentFlags.BoolVar(&noIgnoreFlag, flagNoIgnoreName, false, "Disable use of ignore files")
	persistentFlags.StringVar(&outputFormat, flagFormatName, types.FormatJSON, "Output format (raw or json)")
	persistentFlags.BoolVar(&documentationEnabled, flagDocName, false, "Include documentation entries")

	rootCommand.PersistentPreRunE = func(command *cobra.Command, arguments []string) error {
		if showVersionFlag {
			fmt.Println(versionPrefix, utils.GetApplicationVersion())
			os.Exit(0)
		}
		if outputFormat != types.FormatRaw && outputFormat != types.FormatJSON {
			return fmt.Errorf("Invalid format value '%s'", outputFormat)
		}
		return nil
	}

	rootCommand.AddCommand(newTreeCommand(), newContentCommand(), newCallChainCommand())
	return rootCommand
}

// newTreeCommand creates the tree subcommand.
func newTreeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   types.CommandTree + " [paths]",
		Short: "Display directory structure",
		RunE:  runTreeCommand,
	}
}

// newContentCommand creates the content subcommand.
func newContentCommand() *cobra.Command {
	return &cobra.Command{
		Use:   types.CommandContent + " [paths]",
		Short: "Display file contents",
		RunE:  runContentCommand,
	}
}

// newCallChainCommand creates the callchain subcommand.
func newCallChainCommand() *cobra.Command {
	return &cobra.Command{
		Use:   types.CommandCallChain + " <function>",
		Short: "Display function call chains",
		Args:  cobra.ExactArgs(1),
		RunE:  runCallChainCommand,
	}
}

// runTreeCommand executes the tree command with provided arguments.
func runTreeCommand(command *cobra.Command, arguments []string) error {
	paths := arguments
	if len(paths) == 0 {
		paths = []string{defaultPath}
	}
	if documentationEnabled {
		fmt.Fprintln(os.Stderr, docIgnoredMessage)
	}
	useGitignore := !noGitignoreFlag
	useIgnoreFile := !noIgnoreFlag
	return runTool(types.CommandTree, paths, exclusionFolder, useGitignore, useIgnoreFile, outputFormat, false)
}

// runContentCommand executes the content command with provided arguments.
func runContentCommand(command *cobra.Command, arguments []string) error {
	paths := arguments
	if len(paths) == 0 {
		paths = []string{defaultPath}
	}
	useGitignore := !noGitignoreFlag
	useIgnoreFile := !noIgnoreFlag
	return runTool(types.CommandContent, paths, exclusionFolder, useGitignore, useIgnoreFile, outputFormat, documentationEnabled)
}

// runCallChainCommand executes the callchain command.
func runCallChainCommand(command *cobra.Command, arguments []string) error {
	useGitignore := !noGitignoreFlag
	useIgnoreFile := !noIgnoreFlag
	return runTool(types.CommandCallChain, arguments, exclusionFolder, useGitignore, useIgnoreFile, outputFormat, documentationEnabled)
}

// runTool dispatches commands to their respective handlers.
func runTool(
	command string,
	paths []string,
	exclusionFolder string,
	useGitignore bool,
	useIgnoreFile bool,
	format string,
	documentationEnabled bool,
) error {
	rootPath, _ := os.Getwd()
	var documentationCollector *docs.Collector
	if documentationEnabled {
		collector, collectorError := docs.NewCollector(rootPath)
		if collectorError != nil {
			return collectorError
		}
		documentationCollector = collector
	}

	switch command {
	case types.CommandCallChain:
		return runCallChain(paths[0], format, documentationEnabled, documentationCollector, rootPath)
	case types.CommandTree, types.CommandContent:
		return runTreeOrContentCommand(command, paths, exclusionFolder, useGitignore, useIgnoreFile, format, documentationEnabled, documentationCollector)
	default:
		return fmt.Errorf("unsupported command")
	}
}

// runCallChain renders call chain information for the target function.
func runCallChain(
	target string,
	format string,
	withDocumentation bool,
	collector *docs.Collector,
	root string,
) error {
	data, callChainError := commands.GetCallChainData(target, withDocumentation, collector, root)
	if callChainError != nil {
		return callChainError
	}
	if format == types.FormatJSON {
		rendered, renderError := output.RenderCallChainJSON(data)
		if renderError != nil {
			return renderError
		}
		fmt.Println(rendered)
	} else {
		fmt.Println(output.RenderCallChainRaw(data))
	}
	return nil
}

// runTreeOrContentCommand handles tree and content commands.
func runTreeOrContentCommand(
	command string,
	paths []string,
	exclusionFolder string,
	useGitignore bool,
	useIgnoreFile bool,
	format string,
	withDocumentation bool,
	collector *docs.Collector,
) error {
	validated, validationError := resolveAndValidatePaths(paths)
	if validationError != nil {
		return validationError
	}

	var collected []interface{}
	for _, pathInfo := range validated {
		if pathInfo.IsDir {
			patterns, patternError := config.LoadCombinedIgnorePatterns(pathInfo.AbsolutePath, exclusionFolder, useGitignore, useIgnoreFile)
			if patternError != nil {
				fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", pathInfo.AbsolutePath, patternError)
				continue
			}
			if command == types.CommandTree {
				nodes, nodeError := commands.GetTreeData(pathInfo.AbsolutePath, patterns)
				if nodeError == nil && len(nodes) > 0 {
					collected = append(collected, nodes[0])
				}
			} else {
				files, fileError := commands.GetContentData(pathInfo.AbsolutePath, patterns)
				if fileError == nil {
					for index := range files {
						collected = append(collected, &files[index])
					}
				}
			}
		} else {
			if command == types.CommandTree {
				collected = append(collected, &types.TreeOutputNode{
					Path: pathInfo.AbsolutePath,
					Name: filepath.Base(pathInfo.AbsolutePath),
					Type: types.NodeTypeFile,
				})
			} else {
				data, readError := os.ReadFile(pathInfo.AbsolutePath)
				if readError != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to read %s: %v\n", pathInfo.AbsolutePath, readError)
					continue
				}
				collected = append(collected, &types.FileOutput{
					Path:    pathInfo.AbsolutePath,
					Type:    types.NodeTypeFile,
					Content: string(data),
				})
			}
		}
	}

	var documentationEntries []types.DocumentationEntry
	if withDocumentation && collector != nil {
		for _, item := range collected {
			if fileOutput, isFileOutput := item.(*types.FileOutput); isFileOutput {
				entries, _ := collector.CollectFromFile(fileOutput.Path)
				documentationEntries = append(documentationEntries, entries...)
			}
		}
		sort.Slice(documentationEntries, func(firstIndex, secondIndex int) bool {
			if documentationEntries[firstIndex].Kind != documentationEntries[secondIndex].Kind {
				return documentationEntries[firstIndex].Kind < documentationEntries[secondIndex].Kind
			}
			return documentationEntries[firstIndex].Name < documentationEntries[secondIndex].Name
		})
	}

	if format == types.FormatJSON {
		rendered, renderError := output.RenderJSON(documentationEntries, collected)
		if renderError != nil {
			return renderError
		}
		fmt.Println(rendered)
		return nil
	}

	return output.RenderRaw(command, documentationEntries, collected)
}

// resolveAndValidatePaths turns input paths into validated absolute paths.
func resolveAndValidatePaths(inputs []string) ([]types.ValidatedPath, error) {
	seen := make(map[string]struct{})
	var result []types.ValidatedPath
	for _, inputPath := range inputs {
		absolutePath, absoluteError := filepath.Abs(inputPath)
		if absoluteError != nil {
			return nil, fmt.Errorf("abs failed for '%s': %w", inputPath, absoluteError)
		}
		cleanPath := filepath.Clean(absolutePath)
		if _, alreadySeen := seen[cleanPath]; alreadySeen {
			continue
		}
		info, statError := os.Stat(cleanPath)
		if statError != nil {
			if os.IsNotExist(statError) {
				return nil, fmt.Errorf("path '%s' does not exist", inputPath)
			}
			return nil, fmt.Errorf("stat failed for '%s': %w", inputPath, statError)
		}
		seen[cleanPath] = struct{}{}
		result = append(result, types.ValidatedPath{AbsolutePath: cleanPath, IsDir: info.IsDir()})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid paths")
	}
	return result, nil
}
