# ctx

Ctx is a command‑line tool written in Go that displays a directory tree view, outputs file contents for specified
files and directories, or analyzes the call chain for a given function in the repository. It supports exclusion patterns
via .ignore and .gitignore files within each directory, as well as an optional global exclusion flag. It can output in
raw text format (default) or structured JSON.

## Features

- **Mixed File/Directory Processing:** Accepts one or more file and/or directory paths as input for `tree` and `content`
  commands.
- **Call Chain Analysis:** Analyzes the call graph for a specified function using the `callchain` command. Provide the
  fully qualified (or suffix) function name as the sole argument.
- **Output Formats:** Supports "raw" text output (default) and "json" output using the --format flag for all commands.
- **Tree Command (`tree`, `t`):**
    - *Raw format:* Recursively displays the directory structure for each specified directory in a tree‑like format,
      listing explicitly provided file paths with "[File]".
    - *JSON format:* Outputs a JSON array where each element represents an input path. Directories include a nested "
      children" array.
- **Content Command (`content`, `c`):**
    - *Raw format:* Outputs the content of explicitly provided files and concatenates contents of files within
      directories, separated by headers.
    - *JSON format:* Outputs a JSON array of objects, each containing the "path", "type", and "content" of successfully
      read files.
- **Call Chain Command (`callchain`, `cc`):**
    - *Raw format:* Displays the target function, its callers, and callees, followed by the source code of these
      functions.
    - *JSON format:* Outputs a JSON object with `targetFunction`, `callers`, `callees`, and a `functions` map (name to
      source code).
- **Exclusion Patterns (for `tree` and `content`):**
    - Reads patterns from a .ignore file located at the root of each processed directory (can be disabled with
      --no-ignore).
    - Reads patterns from a .gitignore file located at the root of each processed directory by default (can be disabled
      with --no-gitignore).
    - A global exclusion flag (-e or --e) excludes a designated folder if it appears as a direct child in any specified
      directory.
- **Command Abbreviations:**
    - `t` is an alias for `tree`.
    - `c` is an alias for `content`.
    - `cc` is an alias for `callchain`.
- **Deduplication:** Duplicate input paths (after resolving to absolute paths) are processed only once for `tree` and
  `content`.

## Downloads

For prebuilt binaries and packaged releases, please visit the [Releases](https://github.com/temirov/ctx/releases) page.
Binaries built for macOS (Intel and ARM), Linux (Intel), and Windows (Intel) are available there.

## Installation

1. Ensure you have Go installed (version 1.21 or later recommended).
2. Make sure your GOBIN (or GOPATH/bin) is in your system's PATH.
3. Install the utility:
   ```bash
   go install github.com/temirov/ctx@latest
   ```
4. Verify the installation:
   ```bash
   ctx --help
   ```

## Usage

```bash
ctx <tree|t|content|c|callchain|cc> [arguments...] [flags] [--format <raw|json>] [--version]
```

For the **tree** and **content** commands, arguments are file or directory paths. If none are given, they default to the
current directory.
For the **callchain** command, provide exactly one fully qualified (or suffix) function name present in the repository
as the argument.

### Commands

- **tree (or t):** Displays a directory tree view.
- **content (or c):** Outputs file contents.
- **callchain (or cc):** Analyzes the call graph for a specified function and returns its callers and callees.

### Flags

- **-e or --e _exclusion_folder_:** Specifies a folder name to exclude during directory traversal (for `tree`/
  `content`).
- **--no-gitignore:** Disables loading of .gitignore files (for `tree`/`content`).
- **--no-ignore:** Disables loading of .ignore files (for `tree`/`content`).
- **--format _raw|json_:** Specifies the output format for any command. Default is "raw".
- **--version:** Displays the application version and exits.

### Examples

Display a raw tree view for directories, excluding "dist":

```bash
ctx tree projectA projectB -e dist
```

Output file contents in JSON format:

```bash
ctx content main.go pkg -e logs --format json
```

Analyze the call chain for a function with raw output:

```bash
ctx callchain github.com/temirov/ctx/commands.GetContentData --format raw
```

Analyze the call chain for a function with JSON output:

```bash
ctx callchain github.com/temirov/ctx/commands.GetContentData --format json
```

## Output Formats

- **Raw format:** Outputs human-readable text tailored to the command.
- **JSON format:** Outputs structured JSON suitable for machine processing.

## Configuration (for `tree` and `content`)

Exclusion patterns are loaded from .ignore and .gitignore files located at the root of each processed directory.
Patterns from both files are combined for directory traversal. Ignore rules do not apply to explicitly specified file
arguments.

## License

This project is licensed under the [MIT License](LICENSE).