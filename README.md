# ctx

Ctx is a command-line tool written in Go that displays a directory tree view, outputs file contents for specified
files and directories, or analyzes the call chain for a given function in the repository. It supports exclusion patterns
via .ignore and .gitignore files within each directory, an optional global exclusion flag, configurable output formats,
and **optional embedded documentation** for referenced packages and symbols.

## Features

- **Mixed File/Directory Processing:** Accepts one or more file and/or directory paths as input for `tree` and `content`
  commands.
- **Call Chain Analysis:** Analyzes the call graph for a specified function using the `callchain` command. Provide the
  fully qualified (or suffix) function name as the sole argument. The traversal depth is configurable with `--depth`.
- **Embedded Documentation (`--doc`):** When the `--doc` flag is used with the `content` or `callchain` command,
  ctx embeds documentation for imported third-party packages and referenced functions in the output (both *raw* and
  *json* formats).
- **Output Formats:** Supports **raw** text output (default), **json**, and **xml** output using the `--format` flag for all
  commands.
- **Tree Command (`tree`, `t`):**
    - *Raw format:* Recursively displays the directory structure for each specified directory in a tree-like format,
      listing explicitly provided file paths with `[File]`.
    - *JSON format:* Outputs a JSON array where each element represents an input path. Directories include a nested
      `children` array.
- **Content Command (`content`, `c`):**
    - *Raw format:* Outputs the content of explicitly provided files and concatenates contents of files within
      directories, separated by headers. When `--doc` is used, documentation for imported packages and symbols is
      appended after each file block.
    - *JSON format:* Outputs a JSON array of objects, each containing the `path`, `type`, and `content` of successfully
      read files. Documentation (when `--doc` is used) is included in a `documentation` field.
- **Call Chain Command (`callchain`, `cc`):**
    - *Raw format:* Displays the target function, its callers, and callees, followed by the source code of these
      functions. When `--doc` is used, documentation for referenced external packages/functions is appended.
    - *JSON format:* Outputs a JSON object with `targetFunction`, `callers`, `callees`, a `functions` map (name →
      source), and (when `--doc`) a `documentation` array.
    - *Depth control (`--depth`):* Limits traversal of callers and callees. The default depth is `1`, which yields only
      direct callers and callees. For example:

      ```bash
      ctx callchain github.com/temirov/ctx/internal/commands.GetContentData --depth 2
      ```
- **Exclusion Patterns (for `tree` and `content`):**
    - Reads patterns from a `.ignore` file located at the root of each processed directory (can be disabled with
      `--no-ignore`).
    - Reads patterns from a `.gitignore` file located at the root of each processed directory by default (can be
      disabled with `--no-gitignore`).
    - Skips the `.git` directory unless the `--git` flag is provided.
    - The `[binary]` section in `.ignore` lists patterns whose binary contents are base64-encoded and included in output.
    - A global exclusion flag (`-e` or `--e`) excludes a designated folder if it appears as a direct child in any
      specified directory.
- **Command Abbreviations:**
    - `t` is an alias for `tree`.
    - `c` is an alias for `content`.
    - `cc` is an alias for `callchain`.
- **Deduplication:** Duplicate input paths (after resolving to absolute paths) are processed only once for `tree` and
  `content`.

## Downloads

Pre-built binaries are available on the
[Releases](https://github.com/temirov/ctx/releases) page for macOS (Intel & ARM), Linux (Intel), and Windows (Intel).

## Installation

1. Install Go ≥ 1.21.
2. Add `$GOBIN` (or `$GOPATH/bin`) to your `PATH`.
3. Run:

   ```bash
   go install github.com/temirov/ctx@latest
   ```

4. Verify:

   ```bash
   ctx --help
   ```

## Usage

```bash
ctx <tree|t|content|c|callchain|cc> [arguments...] [flags]
```

### Common Flags

| Flag                  | Applies to         | Description |
|-----------------------|--------------------|---------------------------------------------------------------|
| `-e, --e <folder>`    | tree, content      | Exclude a direct-child folder during directory traversal. |
| `--no-gitignore`      | tree, content      | Disable loading of `.gitignore` files. |
| `--no-ignore`         | tree, content      | Disable loading of `.ignore` files. |
| `--git`               | tree, content      | Include the `.git` directory during traversal. |
| `--format <raw|json|xml>` | all commands       | Select output format (default `raw`). |
| `--doc`               | content, callchain | Embed documentation for referenced external packages and symbols into the output. |
| `--depth <number>`    | callchain          | Limit call graph traversal depth (default `1`). |
| `--version`           | all commands       | Print ctx version and exit. |

### Examples

Display a raw tree view excluding `dist` folders:

```bash
ctx tree projectA projectB -e dist
```

Output file contents in JSON with embedded docs:

```bash
ctx content main.go pkg --doc --format json
```

Analyze the call chain for a function including docs:

```bash
ctx callchain github.com/temirov/ctx/internal/commands.GetContentData --depth 2 --doc --format raw
```

## Output Formats

| Format | tree command                   | content command                            | callchain command |
|--------|--------------------------------|--------------------------------------------|------------------|
| raw    | Text tree view (`[File] path`) | File blocks (`File: path ... End of file`) | Metadata, source blocks; docs when `--doc`. |
| json   | `[]TreeOutputNode`             | `[]FileOutput`                             | `CallChainOutput` |
| xml    | `result/code/item` nodes       | `result/code/item` nodes                   | `callchains/callchain` |

All JSON and XML outputs include a `mimeType` field for every file. Raw output never displays MIME type information.

## Configuration

Exclusion patterns are loaded **only** during directory traversal; explicitly listed file paths are never ignored.

## Binary File Handling

When a binary file is encountered, `ctx` omits its content in raw output. JSON and XML results always include the file's MIME type. This is the default behavior when `.ignore` contains no `[binary]` section and no legacy directives:

```
# .ignore
```

```bash
ctx content image.png --format raw
File: image.png
(binary content omitted)
End of file: image.png
```

To include binary data, add a `[binary]` section to `.ignore` and list matching patterns. Matched files are emitted as base64-encoded strings:

```
[binary]
image.png
```

```bash
ctx content . --format json
[
  {
    "path": "image.png",
    "type": "binary",
    "content": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PdpvJwAAAABJRU5ErkJggg==",
    "mimeType": "image/png"
  }
]
```

## License

ctx is released under the [MIT License](MIT-LICENSE).

