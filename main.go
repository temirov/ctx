package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/temirov/ctx/commands"
	"github.com/temirov/ctx/config"
	"github.com/temirov/ctx/docs"
	"github.com/temirov/ctx/output"
	"github.com/temirov/ctx/types"
	"github.com/temirov/ctx/utils"
)

const (
	flagVersion     = "--version"
	flagExcludeS    = "-e"
	flagExcludeL    = "--e"
	flagNoGitignore = "--no-gitignore"
	flagNoIgnore    = "--no-ignore"
	flagFormat      = "--format"
	flagDoc         = "--doc"
	defaultPath     = "."
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == flagVersion {
			fmt.Println("ctx version:", utils.GetApplicationVersion())
			os.Exit(0)
		}
	}
	command, paths, exclusionFolder, useGitignore, useIgnoreFile, outputFormat, documentationEnabled := parseArgsOrExit()
	if err := runTool(command, paths, exclusionFolder, useGitignore, useIgnoreFile, outputFormat, documentationEnabled); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func parseArgsOrExit() (string, []string, string, bool, bool, string, bool) {
	if len(os.Args) < 2 {
		printUsage()
	}
	useGitignore := true
	useIgnoreFile := true
	outputFormat := types.FormatJSON
	withDoc := false

	rawCommand := os.Args[1]
	var command string
	switch rawCommand {
	case types.CommandTree, "t":
		command = types.CommandTree
	case types.CommandContent, "c":
		command = types.CommandContent
	case types.CommandCallChain, "cc":
		command = types.CommandCallChain
	default:
		fmt.Printf("Error: invalid command '%s'\n", rawCommand)
		printUsage()
	}

	args := os.Args[2:]
	var paths []string
	flagsSeen := false
	exclusionFolder := ""

	for i := 0; i < len(args); {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagsSeen = true
			switch arg {
			case flagExcludeS, flagExcludeL:
				if i+1 >= len(args) {
					fmt.Printf("Error: missing exclusion folder after %s\n", arg)
					printUsage()
				}
				exclusionFolder = args[i+1]
				i += 2
			case flagNoGitignore:
				useGitignore = false
				i++
			case flagNoIgnore:
				useIgnoreFile = false
				i++
			case flagFormat:
				if i+1 >= len(args) {
					fmt.Printf("Error: missing format after %s\n", arg)
					printUsage()
				}
				val := strings.ToLower(args[i+1])
				if val != types.FormatRaw && val != types.FormatJSON {
					fmt.Printf("Invalid format value '%s'\n", val)
					printUsage()
				}
				outputFormat = val
				i += 2
			case flagDoc:
				withDoc = true
				i++
			default:
				fmt.Printf("Error: unknown flag %s\n", arg)
				printUsage()
			}
		} else {
			if flagsSeen && command != types.CommandCallChain {
				fmt.Printf("Error: positional argument '%s' after flags\n", arg)
				printUsage()
			}
			paths = append(paths, arg)
			i++
		}
	}

	if (command == types.CommandTree || command == types.CommandContent) && len(paths) == 0 {
		paths = []string{defaultPath}
	}
	if command == types.CommandCallChain && len(paths) != 1 {
		fmt.Printf("Error: '%s' requires exactly one argument\n", types.CommandCallChain)
		printUsage()
	}
	if command == types.CommandTree && withDoc {
		fmt.Fprintln(os.Stderr, "--doc ignored for tree")
		withDoc = false
	}
	return command, paths, exclusionFolder, useGitignore, useIgnoreFile, outputFormat, withDoc
}

func printUsage() {
	exe := filepath.Base(os.Args[0])
	fmt.Printf("Usage:\n  %s <tree|t|content|c|callchain|cc> [paths] [-e folder] [--no-gitignore] [--no-ignore] [--format raw|json] [--doc]\n", exe)
	os.Exit(1)
}

func runTool(
	command string,
	paths []string,
	exclusionFolder string,
	useGitignore, useIgnoreFile bool,
	format string,
	documentationEnabled bool,
) error {
	root, _ := os.Getwd()
	var collector *docs.Collector
	if documentationEnabled {
		col, err := docs.NewCollector(root)
		if err != nil {
			return err
		}
		collector = col
	}

	switch command {
	case types.CommandCallChain:
		return runCallChain(paths[0], format, documentationEnabled, collector, root)
	case types.CommandTree, types.CommandContent:
		return runTreeOrContentCommand(command, paths, exclusionFolder, useGitignore, useIgnoreFile, format, documentationEnabled, collector)
	default:
		return fmt.Errorf("unsupported command")
	}
}

func runCallChain(
	target, format string,
	withDoc bool,
	collector *docs.Collector,
	root string,
) error {
	data, err := commands.GetCallChainData(target, withDoc, collector, root)
	if err != nil {
		return err
	}
	if format == types.FormatJSON {
		out, err := output.RenderCallChainJSON(data)
		if err != nil {
			return err
		}
		fmt.Println(out)
	} else {
		fmt.Println(output.RenderCallChainRaw(data))
	}
	return nil
}

func runTreeOrContentCommand(
	command string,
	paths []string,
	exclusionFolder string,
	useGitignore, useIgnoreFile bool,
	format string,
	withDoc bool,
	collector *docs.Collector,
) error {
	validated, err := resolveAndValidatePaths(paths)
	if err != nil {
		return err
	}

	var collected []interface{}
	for _, info := range validated {
		if info.IsDir {
			patterns, err := config.LoadCombinedIgnorePatterns(info.AbsolutePath, exclusionFolder, useGitignore, useIgnoreFile, true)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", info.AbsolutePath, err)
				continue
			}
			if command == types.CommandTree {
				nodes, err := commands.GetTreeData(info.AbsolutePath, patterns)
				if err == nil && len(nodes) > 0 {
					collected = append(collected, nodes[0])
				}
			} else {
				files, err := commands.GetContentData(info.AbsolutePath, patterns)
				if err == nil {
					for i := range files {
						collected = append(collected, &files[i])
					}
				}
			}
		} else {
			if command == types.CommandTree {
				collected = append(collected, &types.TreeOutputNode{
					Path: info.AbsolutePath,
					Name: filepath.Base(info.AbsolutePath),
					Type: types.NodeTypeFile,
				})
			} else {
				data, err := os.ReadFile(info.AbsolutePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to read %s: %v\n", info.AbsolutePath, err)
					continue
				}
				collected = append(collected, &types.FileOutput{
					Path:    info.AbsolutePath,
					Type:    types.NodeTypeFile,
					Content: string(data),
				})
			}
		}
	}

	var docsEntries []types.DocumentationEntry
	if withDoc && collector != nil {
		for _, item := range collected {
			if f, ok := item.(*types.FileOutput); ok {
				entries, _ := collector.CollectFromFile(f.Path)
				docsEntries = append(docsEntries, entries...)
			}
		}
		sort.Slice(docsEntries, func(i, j int) bool {
			if docsEntries[i].Kind != docsEntries[j].Kind {
				return docsEntries[i].Kind < docsEntries[j].Kind
			}
			return docsEntries[i].Name < docsEntries[j].Name
		})
	}

	if format == types.FormatJSON {
		out, err := output.RenderJSON(docsEntries, collected)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	return output.RenderRaw(command, docsEntries, collected)
}

func resolveAndValidatePaths(inputs []string) ([]types.ValidatedPath, error) {
	seen := make(map[string]struct{})
	var result []types.ValidatedPath
	for _, p := range inputs {
		absP, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("abs failed for '%s': %w", p, err)
		}
		clean := filepath.Clean(absP)
		if _, ok := seen[clean]; ok {
			continue
		}
		info, err := os.Stat(clean)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("path '%s' does not exist", p)
			}
			return nil, fmt.Errorf("stat failed for '%s': %w", p, err)
		}
		seen[clean] = struct{}{}
		result = append(result, types.ValidatedPath{AbsolutePath: clean, IsDir: info.IsDir()})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid paths")
	}
	return result, nil
}
