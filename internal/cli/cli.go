// Package cli provides the command line interface.
package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/temirov/ctx/internal/commands"
	"github.com/temirov/ctx/internal/config"
	"github.com/temirov/ctx/internal/docs"
	"github.com/temirov/ctx/internal/output"
	"github.com/temirov/ctx/internal/services/clipboard"
	"github.com/temirov/ctx/internal/services/mcp"
	"github.com/temirov/ctx/internal/services/stream"
	"github.com/temirov/ctx/internal/tokenizer"
	"github.com/temirov/ctx/internal/types"
	"github.com/temirov/ctx/internal/utils"
	"golang.org/x/sync/errgroup"
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
	documentationFlagName = "doc"
	contentFlagName       = "content"
	versionFlagName       = "version"
	versionTemplate       = "ctx version: %s\n"
	defaultPath           = "."
	rootUse               = "ctx"
	rootShortDescription  = "ctx command line interface"
	rootLongDescription   = `ctx inspects project structure and source code.
It renders directory trees, shows file content, and analyzes call chains.
Use --format to select raw, json, or xml output. Use --doc to include documentation for supported commands, and --version to print the application version.`
	versionFlagDescription    = "display application version"
	clipboardFlagName         = "clipboard"
	clipboardFlagDescription  = "copy command output to the system clipboard"
	configFlagName            = "config"
	configFlagDescription     = "path to an application configuration file"
	initFlagName              = "init"
	initFlagDescription       = "generate a configuration file (local or global)"
	forceFlagName             = "force"
	forceFlagDescription      = "overwrite configuration if it already exists"
	mcpFlagName               = "mcp"
	mcpFlagDescription        = "run the program as an MCP server"
	mcpListenAddress          = "127.0.0.1:0"
	mcpStartupMessageFormat   = "MCP server listening on %s\n"
	mcpFlagConflictMessage    = "--mcp cannot be combined with subcommands"
	mcpShutdownTimeout        = 5 * time.Second
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
	defaultTokenizerModelName       = "gpt-4o"
	documentationFlagDescription    = "include documentation"
	contentFlagDescription          = "include file content in output"
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
	errorNoValidPaths              = "no valid paths"
	clipboardCopyErrorFormat       = "failed to copy output to clipboard: %w"
	clipboardServiceMissingMessage = "clipboard functionality requested but unavailable"
	configurationLoadErrorFormat   = "failed to load configuration: %w"
	configurationInitSuccessFormat = "configuration written to %s\n"
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
	rootCommand := createRootCommand(clipboard.NewService())
	return rootCommand.Execute()
}

// createRootCommand builds the root Cobra command.
func createRootCommand(clipboardProvider clipboard.Copier) *cobra.Command {
	var showVersion bool
	var clipboardEnabled bool
	var explicitConfigPath string
	var initTarget string
	var forceInit bool
	var applicationConfig config.ApplicationConfiguration
	var configurationLoaded bool
	var runMCP bool

	rootCommand := &cobra.Command{
		Use:          rootUse,
		Short:        rootShortDescription,
		Long:         rootLongDescription,
		SilenceUsage: true,
		RunE: func(command *cobra.Command, arguments []string) error {
			if runMCP {
				if len(arguments) > 0 {
					return fmt.Errorf("%s accepts no arguments", mcpFlagName)
				}
				return startMCPServer(command.Context(), command.OutOrStdout())
			}
			return command.Help()
		},
		PersistentPreRunE: func(command *cobra.Command, arguments []string) error {
			if showVersion {
				fmt.Printf(versionTemplate, utils.GetApplicationVersion())
				os.Exit(0)
			}
			if runMCP && command.Name() != rootUse {
				return fmt.Errorf(mcpFlagConflictMessage)
			}
			if configurationLoaded {
				return nil
			}
			workingDirectory, workingDirectoryError := os.Getwd()
			if workingDirectoryError != nil {
				return fmt.Errorf(workingDirectoryErrorFormat, workingDirectoryError)
			}
			if command.InheritedFlags().Changed(initFlagName) {
				target := config.InitTarget(initTarget)
				if target == "" {
					target = config.InitTargetLocal
				}
				initializedPath, initError := config.InitializeConfiguration(config.InitOptions{
					Target:           target,
					Force:            forceInit,
					WorkingDirectory: workingDirectory,
				})
				if initError != nil {
					return initError
				}
				fmt.Fprintf(command.OutOrStdout(), configurationInitSuccessFormat, initializedPath)
				os.Exit(0)
			}
			loadedConfiguration, loadError := config.LoadApplicationConfiguration(config.LoadOptions{
				WorkingDirectory: workingDirectory,
				ExplicitFilePath: explicitConfigPath,
			})
			if loadError != nil {
				return fmt.Errorf(configurationLoadErrorFormat, loadError)
			}
			applicationConfig = loadedConfiguration
			configurationLoaded = true
			return nil
		},
	}
	rootCommand.PersistentFlags().BoolVar(&showVersion, versionFlagName, false, versionFlagDescription)
	rootCommand.PersistentFlags().BoolVar(&clipboardEnabled, clipboardFlagName, false, clipboardFlagDescription)
	rootCommand.PersistentFlags().StringVar(&explicitConfigPath, configFlagName, "", configFlagDescription)
	rootCommand.PersistentFlags().StringVar(&initTarget, initFlagName, "", initFlagDescription)
	if initFlag := rootCommand.PersistentFlags().Lookup(initFlagName); initFlag != nil {
		initFlag.NoOptDefVal = string(config.InitTargetLocal)
	}
	rootCommand.PersistentFlags().BoolVar(&forceInit, forceFlagName, false, forceFlagDescription)
	rootCommand.PersistentFlags().BoolVar(&runMCP, mcpFlagName, false, mcpFlagDescription)
	rootCommand.AddCommand(
		createTreeCommand(clipboardProvider, &clipboardEnabled, &applicationConfig),
		createContentCommand(clipboardProvider, &clipboardEnabled, &applicationConfig),
		createCallChainCommand(clipboardProvider, &clipboardEnabled, &applicationConfig),
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
	enabled bool
	model   string
}

func (options tokenOptions) toConfig(workingDirectory string) tokenizer.Config {
	return tokenizer.Config{
		Model:            options.model,
		WorkingDirectory: workingDirectory,
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
func createTreeCommand(clipboardProvider clipboard.Copier, clipboardFlag *bool, applicationConfig *config.ApplicationConfiguration) *cobra.Command {
	var pathConfiguration pathOptions
	var outputFormat string = types.FormatJSON
	var summaryEnabled bool = true
	var tokenConfiguration tokenOptions
	tokenConfiguration.model = defaultTokenizerModelName
	var includeContent bool

	treeCommand := &cobra.Command{
		Use:     treeUse,
		Aliases: []string{treeAlias},
		Short:   treeShortDescription,
		Long:    treeLongDescription,
		Example: treeUsageExample,
		Args:    cobra.ArbitraryArgs,
		RunE: func(command *cobra.Command, arguments []string) error {
			if applicationConfig != nil {
				applyStreamConfiguration(command, applicationConfig.Tree, &pathConfiguration, &outputFormat, nil, &summaryEnabled, &includeContent, &tokenConfiguration)
			}
			if len(arguments) == 0 {
				arguments = []string{defaultPath}
			}
			outputFormatLower := strings.ToLower(outputFormat)
			if !isSupportedFormat(outputFormatLower) {
				return fmt.Errorf(invalidFormatMessage, outputFormatLower)
			}
			clipboardEnabledForCommand := clipboardFlag != nil && *clipboardFlag
			if applicationConfig != nil {
				clipboardEnabledForCommand = resolveClipboardDefault(command, clipboardEnabledForCommand, applicationConfig.Tree.Clipboard)
			}
			return runTool(
				command.Context(),
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
				includeContent,
				tokenConfiguration,
				os.Stdout,
				os.Stderr,
				clipboardEnabledForCommand,
				clipboardProvider,
			)
		},
	}

	addPathFlags(treeCommand, &pathConfiguration)
	treeCommand.Flags().StringVar(&outputFormat, formatFlagName, types.FormatJSON, formatFlagDescription)
	treeCommand.Flags().BoolVar(&summaryEnabled, summaryFlagName, true, summaryFlagDescription)
	treeCommand.Flags().BoolVar(&tokenConfiguration.enabled, tokensFlagName, false, tokensFlagDescription)
	treeCommand.Flags().StringVar(&tokenConfiguration.model, modelFlagName, defaultTokenizerModelName, modelFlagDescription)
	treeCommand.Flags().BoolVar(&includeContent, contentFlagName, false, contentFlagDescription)
	return treeCommand
}

// createContentCommand returns the content subcommand.
func createContentCommand(clipboardProvider clipboard.Copier, clipboardFlag *bool, applicationConfig *config.ApplicationConfiguration) *cobra.Command {
	var pathConfiguration pathOptions
	var outputFormat string = types.FormatJSON
	var documentationEnabled bool
	var summaryEnabled bool = true
	var tokenConfiguration tokenOptions
	tokenConfiguration.model = defaultTokenizerModelName
	includeContent := true

	contentCommand := &cobra.Command{
		Use:     contentUse,
		Aliases: []string{contentAlias},
		Short:   contentShortDescription,
		Long:    contentLongDescription,
		Example: contentUsageExample,
		Args:    cobra.ArbitraryArgs,
		RunE: func(command *cobra.Command, arguments []string) error {
			if applicationConfig != nil {
				applyStreamConfiguration(command, applicationConfig.Content, &pathConfiguration, &outputFormat, &documentationEnabled, &summaryEnabled, &includeContent, &tokenConfiguration)
			}
			if len(arguments) == 0 {
				arguments = []string{defaultPath}
			}
			outputFormatLower := strings.ToLower(outputFormat)
			if !isSupportedFormat(outputFormatLower) {
				return fmt.Errorf(invalidFormatMessage, outputFormatLower)
			}
			clipboardEnabledForCommand := clipboardFlag != nil && *clipboardFlag
			if applicationConfig != nil {
				clipboardEnabledForCommand = resolveClipboardDefault(command, clipboardEnabledForCommand, applicationConfig.Content.Clipboard)
			}
			return runTool(
				command.Context(),
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
				includeContent,
				tokenConfiguration,
				os.Stdout,
				os.Stderr,
				clipboardEnabledForCommand,
				clipboardProvider,
			)
		},
	}

	addPathFlags(contentCommand, &pathConfiguration)
	contentCommand.Flags().StringVar(&outputFormat, formatFlagName, types.FormatJSON, formatFlagDescription)
	contentCommand.Flags().BoolVar(&documentationEnabled, documentationFlagName, false, documentationFlagDescription)
	contentCommand.Flags().BoolVar(&summaryEnabled, summaryFlagName, true, summaryFlagDescription)
	contentCommand.Flags().BoolVar(&tokenConfiguration.enabled, tokensFlagName, false, tokensFlagDescription)
	contentCommand.Flags().StringVar(&tokenConfiguration.model, modelFlagName, defaultTokenizerModelName, modelFlagDescription)
	contentCommand.Flags().BoolVar(&includeContent, contentFlagName, true, contentFlagDescription)
	return contentCommand
}

// createCallChainCommand returns the callchain subcommand.
func createCallChainCommand(clipboardProvider clipboard.Copier, clipboardFlag *bool, applicationConfig *config.ApplicationConfiguration) *cobra.Command {
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
			if applicationConfig != nil {
				applyCallChainConfiguration(command, applicationConfig.CallChain, &outputFormat, &callChainDepth, &documentationEnabled)
			}
			outputFormatLower := strings.ToLower(outputFormat)
			if !isSupportedFormat(outputFormatLower) {
				return fmt.Errorf(invalidFormatMessage, outputFormatLower)
			}
			clipboardEnabledForCommand := clipboardFlag != nil && *clipboardFlag
			if applicationConfig != nil {
				clipboardEnabledForCommand = resolveClipboardDefault(command, clipboardEnabledForCommand, applicationConfig.CallChain.Clipboard)
			}
			return runTool(
				command.Context(),
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
				false,
				tokenOptions{},
				os.Stdout,
				os.Stderr,
				clipboardEnabledForCommand,
				clipboardProvider,
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
	commandContext context.Context,
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
	includeContent bool,
	tokenConfiguration tokenOptions,
	outputWriter io.Writer,
	errorWriter io.Writer,
	clipboardEnabled bool,
	clipboardProvider clipboard.Copier,
) error {
	if commandContext == nil {
		commandContext = context.Background()
	}
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
	var tokenModel string
	if tokenConfiguration.enabled {
		createdCounter, resolvedModel, counterError := tokenizer.NewCounter(tokenConfiguration.toConfig(workingDirectory))
		if counterError != nil {
			return counterError
		}
		tokenCounter = createdCounter
		tokenModel = resolvedModel
	}

	if outputWriter == nil {
		outputWriter = os.Stdout
	}
	if errorWriter == nil {
		errorWriter = os.Stderr
	}

	var clipboardBuffer *bytes.Buffer
	if clipboardEnabled {
		if clipboardProvider == nil {
			return errors.New(clipboardServiceMissingMessage)
		}
		clipboardBuffer = &bytes.Buffer{}
		outputWriter = io.MultiWriter(outputWriter, clipboardBuffer)
	}

	switch commandName {
	case types.CommandCallChain:
		if err := runCallChain(paths[0], format, callChainDepth, documentationEnabled, collector, workingDirectory, outputWriter); err != nil {
			return err
		}
	case types.CommandTree, types.CommandContent:
		if err := runStreamCommand(commandContext, commandName, paths, exclusionPatterns, useGitignore, useIgnoreFile, includeGit, format, documentationEnabled, summaryEnabled, includeContent, tokenCounter, tokenModel, collector, outputWriter, errorWriter); err != nil {
			return err
		}
	default:
		return fmt.Errorf(unsupportedCommandMessage)
	}

	if clipboardEnabled && clipboardBuffer != nil {
		if copyErr := clipboardProvider.Copy(clipboardBuffer.String()); copyErr != nil {
			return fmt.Errorf(clipboardCopyErrorFormat, copyErr)
		}
	}

	return nil
}

// runCallChain processes the callchain command for the specified target and depth.
func runCallChain(
	target string,
	format string,
	callChainDepth int,
	withDocumentation bool,
	collector *docs.Collector,
	moduleRoot string,
	outputWriter io.Writer,
) error {
	callChainData, callChainDataError := commands.GetCallChainData(target, callChainDepth, withDocumentation, collector, moduleRoot)
	if callChainDataError != nil {
		return callChainDataError
	}
	if outputWriter == nil {
		outputWriter = os.Stdout
	}
	if format == types.FormatJSON {
		renderedCallChainJSONOutput, renderCallChainJSONError := output.RenderCallChainJSON(callChainData)
		if renderCallChainJSONError != nil {
			return renderCallChainJSONError
		}
		fmt.Fprintln(outputWriter, renderedCallChainJSONOutput)
	} else if format == types.FormatXML {
		renderedCallChainXMLOutput, renderCallChainXMLError := output.RenderCallChainXML(callChainData)
		if renderCallChainXMLError != nil {
			return renderCallChainXMLError
		}
		fmt.Fprintln(outputWriter, renderedCallChainXMLOutput)
	} else {
		fmt.Fprintln(outputWriter, output.RenderCallChainRaw(callChainData))
	}
	return nil
}

// runStreamCommand executes tree or content commands for the given paths.
func runStreamCommand(
	commandContext context.Context,
	commandName string,
	paths []string,
	exclusionPatterns []string,
	useGitignore bool,
	useIgnoreFile bool,
	includeGit bool,
	format string,
	withDocumentation bool,
	withSummary bool,
	includeContent bool,
	tokenCounter tokenizer.Counter,
	tokenModel string,
	collector *docs.Collector,
	outputWriter io.Writer,
	errorWriter io.Writer,
) (err error) {
	if commandContext == nil {
		commandContext = context.Background()
	}
	validatedPaths, pathValidationError := resolveAndValidatePaths(paths)
	if pathValidationError != nil {
		return pathValidationError
	}

	totalRootPaths := len(validatedPaths)

	renderCommandName := commandName
	if includeContent {
		renderCommandName = types.CommandContent
	} else {
		renderCommandName = types.CommandTree
	}

	var renderer output.StreamRenderer
	switch format {
	case types.FormatRaw:
		renderer = output.NewRawStreamRenderer(outputWriter, errorWriter, renderCommandName, withSummary)
	case types.FormatJSON:
		renderer = output.NewJSONStreamRenderer(outputWriter, errorWriter, renderCommandName, totalRootPaths, withSummary, includeContent)
	case types.FormatXML:
		renderer = output.NewXMLStreamRenderer(outputWriter, errorWriter, renderCommandName, totalRootPaths, withSummary, includeContent)
	default:
		return fmt.Errorf(invalidFormatMessage, format)
	}

	defer func() {
		if renderer == nil {
			return
		}
		if flushErr := renderer.Flush(); flushErr != nil && err == nil {
			err = flushErr
		}
	}()

	for _, info := range validatedPaths {
		streamErr := runStreamPath(commandContext, renderer, info, exclusionPatterns, useGitignore, useIgnoreFile, includeGit, includeContent, withDocumentation, tokenCounter, tokenModel, collector)
		if streamErr != nil && !errors.Is(streamErr, context.Canceled) {
			if errorWriter != nil {
				fmt.Fprintf(errorWriter, warningSkipPathFormat, info.AbsolutePath, streamErr)
			}
		}
	}

	return nil
}

func runStreamPath(
	ctx context.Context,
	renderer output.StreamRenderer,
	path types.ValidatedPath,
	exclusionPatterns []string,
	useGitignore bool,
	useIgnoreFile bool,
	includeGit bool,
	includeContent bool,
	withDocumentation bool,
	tokenCounter tokenizer.Counter,
	tokenModel string,
	collector *docs.Collector,
) error {
	var ignorePatterns []string
	var binaryPatterns []string
	if path.IsDir {
		patterns, binary, loadErr := config.LoadRecursiveIgnorePatterns(path.AbsolutePath, exclusionPatterns, useGitignore, useIgnoreFile, includeGit)
		if loadErr != nil {
			return loadErr
		}
		ignorePatterns = patterns
		if includeContent {
			binaryPatterns = binary
		}
	}

	producer := func(streamCtx context.Context, ch chan<- stream.Event) error {
		options := stream.TreeOptions{
			Root:                  path.AbsolutePath,
			IgnorePatterns:        ignorePatterns,
			TokenCounter:          tokenCounter,
			TokenModel:            tokenModel,
			IncludeContent:        includeContent,
			BinaryContentPatterns: binaryPatterns,
		}
		return stream.StreamTree(streamCtx, options, ch)
	}

	consumer := func(event stream.Event) error {
		if withDocumentation && collector != nil && event.Kind == stream.EventKindFile && event.File != nil {
			if entries, docErr := collector.CollectFromFile(event.File.Path); docErr == nil && len(entries) > 0 {
				event.File.Documentation = entries
			}
		}
		return renderer.Handle(event)
	}

	return dispatchStream(ctx, producer, consumer)
}

func dispatchStream(
	ctx context.Context,
	produce func(context.Context, chan<- stream.Event) error,
	consume func(stream.Event) error,
) error {
	group, streamCtx := errgroup.WithContext(ctx)
	events := make(chan stream.Event)

	group.Go(func() error {
		defer close(events)
		return produce(streamCtx, events)
	})

	group.Go(func() error {
		for {
			select {
			case <-streamCtx.Done():
				return streamCtx.Err()
			case event, ok := <-events:
				if !ok {
					return nil
				}
				if err := consume(event); err != nil {
					return err
				}
			}
		}
	})

	if err := group.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func applyStreamConfiguration(
	command *cobra.Command,
	configuration config.StreamCommandConfiguration,
	paths *pathOptions,
	outputFormat *string,
	documentationEnabled *bool,
	summaryEnabled *bool,
	includeContent *bool,
	tokens *tokenOptions,
) {
	if command == nil {
		return
	}
	if configuration.Format != "" && outputFormat != nil && !command.Flags().Changed(formatFlagName) {
		*outputFormat = configuration.Format
	}
	if configuration.Summary != nil && summaryEnabled != nil && !command.Flags().Changed(summaryFlagName) {
		*summaryEnabled = *configuration.Summary
	}
	if configuration.Documentation != nil && documentationEnabled != nil && !command.Flags().Changed(documentationFlagName) {
		*documentationEnabled = *configuration.Documentation
	}
	if configuration.IncludeContent != nil && includeContent != nil && !command.Flags().Changed(contentFlagName) {
		*includeContent = *configuration.IncludeContent
	}
	if tokens != nil {
		if configuration.Tokens.Enabled != nil && !command.Flags().Changed(tokensFlagName) {
			tokens.enabled = *configuration.Tokens.Enabled
		}
		if configuration.Tokens.Model != "" && !command.Flags().Changed(modelFlagName) {
			tokens.model = configuration.Tokens.Model
		}
	}
	if paths != nil {
		if len(configuration.Paths.Exclude) > 0 && !command.Flags().Changed(exclusionFlagName) {
			paths.exclusionPatterns = append([]string{}, configuration.Paths.Exclude...)
		}
		if configuration.Paths.UseGitignore != nil && !command.Flags().Changed(noGitignoreFlagName) {
			paths.disableGitignore = !*configuration.Paths.UseGitignore
		}
		if configuration.Paths.UseIgnoreFile != nil && !command.Flags().Changed(noIgnoreFlagName) {
			paths.disableIgnoreFile = !*configuration.Paths.UseIgnoreFile
		}
		if configuration.Paths.IncludeGit != nil && !command.Flags().Changed(includeGitFlagName) {
			paths.includeGit = *configuration.Paths.IncludeGit
		}
	}
}

func applyCallChainConfiguration(
	command *cobra.Command,
	configuration config.CallChainConfiguration,
	outputFormat *string,
	depth *int,
	documentationEnabled *bool,
) {
	if command == nil {
		return
	}
	if configuration.Format != "" && outputFormat != nil && !command.Flags().Changed(formatFlagName) {
		*outputFormat = configuration.Format
	}
	if configuration.Depth != nil && depth != nil && !command.Flags().Changed(callChainDepthFlagName) {
		*depth = *configuration.Depth
	}
	if configuration.Documentation != nil && documentationEnabled != nil && !command.Flags().Changed(documentationFlagName) {
		*documentationEnabled = *configuration.Documentation
	}
}

func resolveClipboardDefault(command *cobra.Command, cliValue bool, configurationValue *bool) bool {
	if command == nil || configurationValue == nil {
		return cliValue
	}
	inherited := command.InheritedFlags()
	if inherited != nil && inherited.Changed(clipboardFlagName) {
		return cliValue
	}
	return *configurationValue
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

func mcpCapabilities() []mcp.Capability {
	return []mcp.Capability{
		{Name: types.CommandTree, Description: treeShortDescription},
		{Name: types.CommandContent, Description: contentShortDescription},
		{Name: types.CommandCallChain, Description: callchainShortDescription},
	}
}

func startMCPServer(parent context.Context, output io.Writer) error {
	ctx, cancel := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	writer := output
	if writer == nil {
		writer = os.Stdout
	}

	workingDirectory, workingDirectoryError := os.Getwd()
	if workingDirectoryError != nil {
		return fmt.Errorf(workingDirectoryErrorFormat, workingDirectoryError)
	}

	server := mcp.NewServer(mcp.Config{
		Address:         mcpListenAddress,
		Capabilities:    mcpCapabilities(),
		Executors:       mcpCommandExecutors(),
		RootDirectory:   workingDirectory,
		ShutdownTimeout: mcpShutdownTimeout,
	})

	notify := func(address string) {
		fmt.Fprintf(writer, mcpStartupMessageFormat, address)
	}

	if err := server.Run(ctx, notify); err != nil {
		return fmt.Errorf("run MCP server: %w", err)
	}
	return nil
}
