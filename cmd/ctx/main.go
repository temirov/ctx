package main

import (
	"fmt"
	"log"
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
	rootShortDescription       = "ctx CLI"
	treeUse                    = "tree [paths...]"
	contentUse                 = "content [paths...]"
	callchainUse               = "callchain <function>"
	treeShortDescription       = "display directory tree"
	contentShortDescription    = "show file contents"
	callchainShortDescription  = "analyze call chains"
	callChainDepthFlagName     = "depth"
	unsupportedCommandMessage  = "unsupported command"
	defaultCallChainDepth      = 1
	callChainDepthDescription  = "traversal depth"
)

// main is the entry point of the application.
func main() {
	rootCommand := createRootCommand()
	if err := rootCommand.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// createRootCommand builds the root Cobra command.
func createRootCommand() *cobra.Command {
	var showVersion bool

	rootCommand := &cobra.Command{
		Use:          rootUse,
		Short:        rootShortDescription,
		SilenceUsage: true,
		Run:          func(command *cobra.Command, arguments []string) {},
		PersistentPreRun: func(command *cobra.Command, arguments []string) {
			if showVersion {
				fmt.Printf(versionTemplate, utils.GetApplicationVersion())
				os.Exit(0)
			}
		},
	}
	rootCommand.PersistentFlags().BoolVar(&showVersion, versionFlagName, false, "display application version")
	rootCommand.AddCommand(
		createTreeCommand(),
		createContentCommand(),
		createCallChainCommand(),
	)
	return rootCommand
}

// createTreeCommand returns the tree subcommand.
func createTreeCommand() *cobra.Command {
	var exclusionFolder string
	var disableGitignore bool
	var disableIgnoreFile bool
	var includeGit bool
	var outputFormat string = types.FormatJSON
	var withDocumentation bool

	treeCommand := &cobra.Command{
		Use:     treeUse,
		Aliases: []string{"t"},
		Short:   treeShortDescription,
		Args:    cobra.ArbitraryArgs,
		RunE: func(command *cobra.Command, arguments []string) error {
			if len(arguments) == 0 {
				arguments = []string{defaultPath}
			}
			if withDocumentation {
				fmt.Fprintln(os.Stderr, documentationIgnoredNotice)
				withDocumentation = false
			}
			outputFormatLower := strings.ToLower(outputFormat)
			if outputFormatLower != types.FormatRaw && outputFormatLower != types.FormatJSON && outputFormatLower != types.FormatXML {
				return fmt.Errorf("Invalid format value '%s'", outputFormatLower)
			}
			return runTool(
				types.CommandTree,
				arguments,
				exclusionFolder,
				!disableGitignore,
				!disableIgnoreFile,
				includeGit,
				defaultCallChainDepth,
				outputFormatLower,
				withDocumentation,
			)
		},
	}

	treeCommand.Flags().StringVarP(&exclusionFolder, exclusionFlagName, exclusionFlagName, "", "exclude folder")
	treeCommand.Flags().BoolVar(&disableGitignore, noGitignoreFlagName, false, "do not use .gitignore")
	treeCommand.Flags().BoolVar(&disableIgnoreFile, noIgnoreFlagName, false, "do not use .ignore")
	treeCommand.Flags().BoolVar(&includeGit, includeGitFlagName, false, "include git directory")
	treeCommand.Flags().StringVar(&outputFormat, formatFlagName, types.FormatJSON, "output format")
	treeCommand.Flags().BoolVar(&withDocumentation, documentationFlagName, false, "include documentation")
	return treeCommand
}

// createContentCommand returns the content subcommand.
func createContentCommand() *cobra.Command {
	var exclusionFolder string
	var disableGitignore bool
	var disableIgnoreFile bool
	var includeGit bool
	var outputFormat string = types.FormatJSON
	var withDocumentation bool

	contentCommand := &cobra.Command{
		Use:     contentUse,
		Aliases: []string{"c"},
		Short:   contentShortDescription,
		Args:    cobra.ArbitraryArgs,
		RunE: func(command *cobra.Command, arguments []string) error {
			if len(arguments) == 0 {
				arguments = []string{defaultPath}
			}
			outputFormatLower := strings.ToLower(outputFormat)
			if outputFormatLower != types.FormatRaw && outputFormatLower != types.FormatJSON && outputFormatLower != types.FormatXML {
				return fmt.Errorf("Invalid format value '%s'", outputFormatLower)
			}
			return runTool(
				types.CommandContent,
				arguments,
				exclusionFolder,
				!disableGitignore,
				!disableIgnoreFile,
				includeGit,
				defaultCallChainDepth,
				outputFormatLower,
				withDocumentation,
			)
		},
	}

	contentCommand.Flags().StringVarP(&exclusionFolder, exclusionFlagName, exclusionFlagName, "", "exclude folder")
	contentCommand.Flags().BoolVar(&disableGitignore, noGitignoreFlagName, false, "do not use .gitignore")
	contentCommand.Flags().BoolVar(&disableIgnoreFile, noIgnoreFlagName, false, "do not use .ignore")
	contentCommand.Flags().BoolVar(&includeGit, includeGitFlagName, false, "include git directory")
	contentCommand.Flags().StringVar(&outputFormat, formatFlagName, types.FormatJSON, "output format")
	contentCommand.Flags().BoolVar(&withDocumentation, documentationFlagName, false, "include documentation")
	return contentCommand
}

// createCallChainCommand returns the callchain subcommand.
func createCallChainCommand() *cobra.Command {
	var outputFormat string = types.FormatJSON
	var withDocumentation bool
	var callChainDepth int = defaultCallChainDepth

	callChainCommand := &cobra.Command{
		Use:     callchainUse,
		Aliases: []string{"cc"},
		Short:   callchainShortDescription,
		Args:    cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			outputFormatLower := strings.ToLower(outputFormat)
			if outputFormatLower != types.FormatRaw && outputFormatLower != types.FormatJSON && outputFormatLower != types.FormatXML {
				return fmt.Errorf("Invalid format value '%s'", outputFormatLower)
			}
			return runTool(
				types.CommandCallChain,
				[]string{arguments[0]},
				"",
				true,
				true,
				false,
				callChainDepth,
				outputFormatLower,
				withDocumentation,
			)
		},
	}

	callChainCommand.Flags().StringVar(&outputFormat, formatFlagName, types.FormatJSON, "output format")
	callChainCommand.Flags().BoolVar(&withDocumentation, documentationFlagName, false, "include documentation")
	callChainCommand.Flags().IntVar(&callChainDepth, callChainDepthFlagName, defaultCallChainDepth, callChainDepthDescription)
	return callChainCommand
}

// runTool executes the command with the provided configuration including call chain depth.
func runTool(
	commandName string,
	paths []string,
	exclusionFolder string,
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
		return runTreeOrContentCommand(commandName, paths, exclusionFolder, useGitignore, useIgnoreFile, includeGit, format, documentationEnabled, collector)
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
	exclusionFolder string,
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
			ignorePatternList, binaryContentPatternList, loadError := config.LoadRecursiveIgnorePatterns(info.AbsolutePath, exclusionFolder, useGitignore, useIgnoreFile, includeGit)
			if loadError != nil {
				fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", info.AbsolutePath, loadError)
				continue
			}
			if commandName == types.CommandTree {
				nodes, dataError := commands.GetTreeData(info.AbsolutePath, ignorePatternList, binaryContentPatternList)
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
				nodeType := types.NodeTypeFile
				mimeType := ""
				if utils.IsFileBinary(info.AbsolutePath) {
					nodeType = types.NodeTypeBinary
					mimeType = utils.DetectMimeType(info.AbsolutePath)
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
					fmt.Fprintf(os.Stderr, "Warning: failed to read %s: %v\n", info.AbsolutePath, fileReadError)
					continue
				}
				fileType := types.NodeTypeFile
				content := string(fileBytes)
				mimeType := ""
				if utils.IsBinary(fileBytes) {
					fileType = types.NodeTypeBinary
					content = ""
					mimeType = utils.DetectMimeType(info.AbsolutePath)
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
