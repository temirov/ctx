# ctx

[![GitHub release](https://img.shields.io/github/release/temirov/ctx.svg)](https://github.com/temirov/ctx/releases)

Ctx is a command-line tool written in Go that displays a directory tree view, outputs file contents for specified
files and directories, or analyzes the call chain for a given function in the repository. It supports exclusion patterns
via .ignore and .gitignore files within each directory, an optional global exclusion flag, configurable output formats,
and **optional embedded documentation** for referenced packages and symbols.

## Quick Start

Install and try ctx's core features: directory trees, file contents with optional documentation, and call-chain analysis.

1. Install:

   ```bash
   go install github.com/temirov/ctx@latest
   ```

2. Show a directory tree:

   ```bash
   ctx tree . --format raw
   ```

3. View file content with docs:

   ```bash
   ctx content main.go --doc
   ```

4. Analyze a call chain:

   ```bash
   ctx callchain github.com/temirov/ctx/internal/commands.GetContentData
   ```

## Features

- **Mixed File/Directory Processing:** Accepts one or more file and/or directory paths as input for `tree` and `content`
  commands.
- **Call Chain Analysis:** Analyzes the call graph for a specified function using the `callchain` command. Provide the
  fully qualified (or suffix) function name as the sole argument. The traversal depth is configurable with `--depth`.
- **Embedded Documentation (`--doc`):** When the `--doc` flag is used with the `content` or `callchain` command,
  ctx embeds documentation for imported third-party packages and referenced functions in the output (*json*, *xml*, and
  *raw* formats).
- **Output Formats:** Supports **json** output (default), **raw** text, and **xml** output using the `--format` flag for all
  commands.
- **Tree Command (`tree`, `t`):**
    - *JSON format:* Outputs a JSON array where each element represents an input path. Directories include a nested
      `children` array.
    - *Raw format:* Recursively displays the directory structure for each specified directory in a tree-like format,
      listing explicitly provided file paths with `[File]`.
    - *XML format:* Produces `result/code/item` nodes capturing the tree structure.
- **Content Command (`content`, `c`):**
    - *JSON format:* Outputs a JSON array of objects, each containing the `path`, `type`, and `content` of successfully
      read files. Documentation (when `--doc` is used) is included in a `documentation` field.
    - *Raw format:* Outputs the content of explicitly provided files and concatenates contents of files within
      directories, separated by headers. When `--doc` is used, documentation for imported packages and symbols is
      appended after each file block.
    - *XML format:* Emits `result/code/item` nodes that mirror the JSON structure, including optional documentation.
- **Call Chain Command (`callchain`, `cc`):**
    - *JSON format:* Outputs a JSON object with `targetFunction`, `callers`, `callees`, a `functions` map (name →
      source), and (when `--doc`) a `documentation` array.
    - *Raw format:* Displays the target function, its callers, and callees, followed by the source code of these
      functions. When `--doc` is used, documentation for referenced external packages/functions is appended.
    - *XML format:* Generates `<callchains><callchain>` elements containing the target, callers, callees, and
      optional documentation.
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
    - A repeatable exclusion flag (`-e` or `--e`) skips paths matching the supplied patterns.
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
| `-e, --e <pattern>`   | tree, content      | Exclude paths matching the pattern; repeat for multiple patterns. |
| `--no-gitignore`      | tree, content      | Disable loading of `.gitignore` files. |
| `--no-ignore`         | tree, content      | Disable loading of `.ignore` files. |
| `--git`               | tree, content      | Include the `.git` directory during traversal. |
| `--format <raw|json|xml>` | all commands       | Select output format (default `json`). |
| `--summary`           | tree, content      | Print total file count and combined size for results (enabled by default, set to `false` to disable). |
| `--tokens`            | tree, content      | Estimate token counts for files and surface totals in summaries. |
| `--model <name>`      | tree, content      | Select tokenizer model (default `gpt-4o`). |
| `--doc`               | content, callchain | Embed documentation for referenced external packages and symbols into the output. |
| `--depth <number>`    | callchain          | Limit call graph traversal depth (default `1`). |
| `--version`           | all commands       | Print ctx version and exit. |

### Examples

Display a raw tree view excluding `dist` folders:

```bash
ctx tree projectA projectB -e dist --format raw
```

Output file contents with embedded docs (JSON by default):

```bash
ctx content main.go pkg --doc
```

Analyze the call chain for a function in XML including docs:

```bash
ctx callchain github.com/temirov/ctx/internal/commands.GetContentData --depth 2 --doc --format xml
```

## Output Formats

| Format | tree command                   | content command                            | callchain command |
|--------|--------------------------------|--------------------------------------------|------------------|
| raw    | Text tree view (`[File] path`) | File blocks (`File: path ... End of file`) | Metadata, source blocks; docs when `--doc`. |
| json   | `[]TreeOutputNode`             | `[]FileOutput`                             | `CallChainOutput` |
| xml    | `result/code/item` nodes       | `result/code/item` nodes                   | `callchains/callchain` |

When `--summary` (enabled by default) is active for tree or content, raw output prepends a `Summary: …` line and shows per-directory totals inside the tree view, while JSON and XML attach `totalFiles`, `totalSize`, and `totalTokens` fields directly to directory entries. The totals are recursive: directory nodes carry the combined size, token count, and file count of everything beneath them, respecting ignore rules and explicit excludes. Pass `--summary false` to suppress these aggregates.

All JSON and XML outputs include a `mimeType` field for every file. Raw output never displays MIME type information.

### Token Counting

Enable `--tokens` to populate a `tokens` field on files (along with a `model` that identifies the tokenizer) and a `totalTokens` aggregate on directories when summaries are included. By default ctx uses OpenAI's `gpt-4o` tokenizer via `tiktoken-go`. Switch models with `--model`; when requesting Anthropic (`claude-*`) or Llama (`llama-*`) models, ctx launches the embedded helpers with [`uv`](https://github.com/astral-sh/uv). Ensure `uv` is available on your `PATH` (or point `CTX_UV` at the executable) and Python 3.11+ will be provisioned automatically. Claude helpers call Anthropic's `messages.count_tokens` endpoint (free to use) and require `ANTHROPIC_API_KEY` to be exported. Llama helpers download a compatible SentencePiece model on demand.

#### Supported models

`ctx` selects the tokenizer backend based on the `--model` prefix:

| Prefix | Backend | Notes |
|--------|---------|-------|
| `gpt-`, `text-embedding`, `davinci`, `curie`, `babbage`, `ada`, `code-` | OpenAI via `tiktoken-go` | Falls back to `cl100k_base` when an exact encoding is unavailable. |
| `claude-` | Anthropic helper (via uv) | Requires `uv`, Python 3.11+, and `ANTHROPIC_API_KEY`; uses Anthropic's `messages.count_tokens` endpoint. Override the executable with `CTX_UV`. |
| `llama-` | SentencePiece helper (via uv) | Requires `uv`, Python 3.11+, and will download a tokenizer model automatically (override with `CTX_SPM_MODEL`). |
| anything else | default (`cl100k_base`) | Safe fallback when no specific tokenizer is known. |

Common Claude shorthands are resolved automatically; for example `--model claude-4.5` maps to `claude-sonnet-4-5-20250929`, and `claude-4` selects `claude-sonnet-4-20250514`.

Example:

```bash
ctx tree . --tokens --summary
```

The summary line now reports total tokens (and the tokenizer model when applicable) alongside file counts and sizes, and each file entry includes its estimated token usage and model in JSON and XML output.

#### Testing Python helpers

To exercise the embedded Python helpers, install the required packages (for example, `pip install anthropic sentencepiece`) and run:

```bash
CTX_TEST_PYTHON=python3 go test -tags python_helpers ./internal/tokenizer
```

Set `CTX_TEST_RUN_HELPERS=1` to enable the optional helper suite and `CTX_TEST_UV` to the uv executable (defaults to `uv`). Tests automatically skip when prerequisites are missing.

#### Streaming pipeline

Tree and content commands now emit structured events from `internal/services/stream`, and format-specific renderers in `internal/output` stream those events directly:

- `--format raw` prints human-friendly lines for every directory, file, chunk, and summary as soon as the event arrives.
- `--format json` writes newline-delimited JSON objects (one per event) that match the schema in `internal/services/stream/events.go`.
- `--format xml` wraps the same event feed in an `<events>` envelope, emitting `<event …>` elements incrementally.

Downstream tools can rebuild aggregated views (trees, summaries, token counts) by consuming the event feed, and the CLI no longer buffers entire directory trees before producing output.

All renderers consume the same event stream; JSON and XML remain schema-compatible with their legacy batch outputs while raw preserves the human-readable format.

Run the streaming regression tests with:

```bash
go test ./internal/services/stream ./internal/output ./internal/cli
```

These tests cover event ordering, renderer behaviour, and CLI integration to ensure events arrive in order and summaries are emitted only after all file content has streamed.

## Configuration

Exclusion patterns are loaded **only** during directory traversal; explicitly listed file paths are never ignored.

> ⚠️ When specifying wildcard patterns (e.g., `-e go.*`), quote them to prevent your shell from expanding the glob before `ctx` runs: `ctx content -e 'go.*'`. The CLI handles pattern matching internally and expects the literal expression.

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
ctx content .
[
  {
    "path": "image.png",
    "type": "binary",
    "content": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PdpvJwAAAABJRU5ErkJggg==",
    "mimeType": "image/png"
  }
]
```

## Releasing

To publish a new version:

1. Update `CHANGELOG.md` with a new section describing the release.
2. Commit the change.
3. Tag the commit and push both the branch and tag:

   ```bash
   git tag vX.Y.Z
   git push origin master
   git push origin vX.Y.Z
   ```

Tags that begin with `v` trigger the release workflow, which builds binaries and uses the matching changelog section as release notes.

## License

ctx is released under the [MIT License](MIT-LICENSE).
