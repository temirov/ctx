# ctx

[![GitHub release](https://img.shields.io/github/release/temirov/ctx.svg)](https://github.com/tyemirov/ctx/releases)

`ctx` is a command-line tool written in Go that displays a directory tree view, outputs file contents for specified
files and directories, or analyzes the call chain for a given function in the repository. It supports exclusion patterns
via .ignore and .gitignore files within each directory, an optional global exclusion flag, configurable output formats,
and **optional embedded documentation** for referenced packages and symbols.

## Quick Start

Install and try ctx's core features: directory trees, file contents with optional documentation, and call-chain analysis.

1. Install:

   ```shell
   go install github.com/tyemirov/ctx@latest
   ```

2. Show a directory tree:

   ```shell
   ctx tree . --format raw
   ```

3. View file content with docs:

   ```shell
   ctx content main.go --doc
   ```

4. Analyze a call chain:

   ```shell
   ctx callchain github.com/tyemirov/ctx/internal/commands.GetContentData
   ```

## Features

- **Mixed File/Directory Processing:** Accepts one or more file and/or directory paths as input for `tree` and `content`
  commands.
- **Call Chain Analysis:** Analyzes the call graph for a specified function using the `callchain` command. Provide the
  fully qualified (or suffix) function name as the sole argument. The traversal depth is configurable with `--depth`.
- **GitHub Documentation Command (`doc`, `d`):** Retrieves and renders documentation stored in GitHub repositories.
  Combine repository coordinates or a GitHub URL with optional rules to clean remote content, and use `--doc` to select
  `relevant` or `full` extraction.
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

      ```shell
      ctx callchain github.com/tyemirov/ctx/internal/commands.GetContentData --depth 2
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
[Releases](https://github.com/tyemirov/ctx/releases) page for macOS (Intel & ARM), Linux (Intel), and Windows (Intel).

## Installation

1. Install Go ≥ 1.21.
2. Add `$GOBIN` (or `$GOPATH/bin`) to your `PATH`.
3. Run:

   ```shell
   go install github.com/tyemirov/ctx@latest
   ```

4. Verify:

   ```shell
   ctx --help
   ```

## Usage

```shell
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
| `--doc`               | content, callchain, doc | Select documentation mode (`disabled`, `relevant`, `full`). `relevant` embeds local symbol docs; `full` also incorporates GitHub content. |
| `--docs-attempt`      | content, callchain | When paired with `--doc full`, heuristically fetch documentation for imported GitHub modules. |
| `--depth <number>`    | callchain          | Limit call graph traversal depth (default `1`). |
| `--version`           | all commands       | Print ctx version and exit. |
| `--copy`              | tree, content, callchain | Copy command output to the system clipboard after rendering completes. |
| `--mcp`               | root command       | Run an HTTP server that publishes ctx capabilities for MCP clients. |

Boolean flags accept any of `true`, `false`, `yes`, `no`, `on`, `off`, `1`, or `0` (case-insensitive), so commands such as `--summary no` or `--copy off` work consistently across the CLI.

When `--copy` is active the command writes its output to the terminal
and copies the same text into the system clipboard. Configure persistent clipboard behaviour by setting `copy: true` under the relevant command (`tree`, `content`, or `callchain`) in `config.yaml`. If clipboard integration
is requested but not supported on the current platform, the command fails with
an explanatory error.

### Examples

Display a raw tree view excluding `dist` folders:

```shell
ctx tree projectA projectB -e dist --format raw
```

Output file contents with embedded docs (JSON by default):

```shell
ctx content main.go pkg --doc
```

Analyze the call chain for a function in XML including docs:

```shell
ctx callchain github.com/tyemirov/ctx/internal/commands.GetContentData --depth 2 --doc --format xml
```

Retrieve GitHub documentation for a project section:

```shell
ctx doc --path jspreadsheet/ce/docs/jspreadsheet --doc full
```

#### Doc Command Parameters

`ctx doc` expects enough information to identify the GitHub documentation directory.

- **Required:** Provide a single `--path` value using `owner/repo[/path]` coordinates or paste a `https://github.com/owner/repo/tree/...` URL. Supplying only `owner/repo` fetches documentation from the repository root.
- **Optional:** `--ref` selects a branch, tag, or commit; `--rules` applies a cleanup rule set; `--doc` toggles the rendered documentation mode; `--copy` copies the rendered output. Combine these with configuration defaults as needed.
- **Heuristics:** Use `--docs-attempt` with `--doc full` on `content` or `callchain` to let ctx detect third-party Go module imports, infer GitHub repositories, and retrieve `docs/` content when available. The same GitHub fetcher powers both the doc command and docs attempts.

Run `ctx doc --help` to review the command description, flag roles, and examples.

### MCP Server Mode

`ctx` can expose its capabilities over HTTP for Model Context Protocol (MCP)
clients. The root command owns the `--mcp` flag; combine it with any other flag
or configuration you normally use with `ctx`:

```shell
ctx --mcp
```

When the server starts it binds to an ephemeral loopback address and prints the
endpoint so clients know where to connect:

```text
MCP server listening on http://127.0.0.1:45873
```

The root path responds with `200 OK` and acts as a basic health check:

```shell
curl --head http://127.0.0.1:45873/
```

Query `/capabilities` to retrieve the advertised commands. The response lists
the command name and the same short description shown in the CLI help output:

```shell
curl http://127.0.0.1:45873/capabilities | jq
```

```json
{
  "capabilities": [
    {
      "name": "tree",
      "description": "Display directory tree as JSON. Paths must be absolute or resolved relative to the reported root directory. Flags: summary (bool), exclude (string[]), includeContent (bool), useGitignore (bool), useIgnore (bool), tokens (bool), model (string), includeGit (bool)."
    },
    {
      "name": "content",
      "description": "Show file contents as JSON. Paths must be absolute or resolved relative to the reported root directory. Flags: summary (bool), documentation (bool), includeContent (bool), exclude (string[]), useGitignore (bool), useIgnore (bool), tokens (bool), model (string), includeGit (bool)."
    },
    {
      "name": "callchain",
      "description": "Analyze Go/Python/JavaScript call chains as JSON. Target must be fully qualified or resolvable in the project. Flags: depth (int), documentation (bool)."
    }
  ]
}
```

Invoke commands by POSTing JSON payloads to `/commands/<name>`. The body mirrors
the corresponding CLI flags. For example, request a tree without summaries:

```shell
curl -X POST http://127.0.0.1:45873/commands/tree \
  -H 'Content-Type: application/json' \
  -d '{"paths":["."],"summary":false}'
```

Successful responses echo the rendered output, the chosen format, and any
warnings emitted during processing:

```json
{
  "output": "{\"path\":\".\",\"name\":\".\",\"type\":\"directory\",\"children\":[],\"totalFiles\":0,\"totalSize\":\"0b\"}\n",
  "format": "json",
  "warnings": []
}
```

Set `"documentation": true` when calling the `content` command to include
documented symbols for Go, JavaScript, and Python files in the JSON response.
MCP endpoints always emit JSON; requests for other output formats are ignored.

Agents should resolve all paths relative to the server's working directory.
Query `/environment` to retrieve that directory before issuing command
requests:

```shell
curl http://127.0.0.1:45873/environment
```

Only absolute paths (or relative paths resolved against the reported root)
should be passed to MCP commands.

#### Registering MCP clients

Start the server inside the project you want to expose:

```shell
ctx --mcp
```

The examples below assume `/Users/alex/src/project` is the root returned by
`/environment` and that `ctx` is on your `$PATH`.

**Claude Desktop** (macOS/Windows): edit
`~/Library/Application Support/Claude/claude_desktop_config.json` and add an
entry under `mcpServers`:

```json
{
  "mcpServers": {
    "ctx": {
      "command": "/usr/local/bin/ctx",
      "args": ["--mcp"],
      "cwd": "/Users/alex/src/project"
    }
  }
}
```

Restart Claude Desktop and the assistant will discover the `ctx` capabilities.

**Codex CLI**: register the server so Codex can proxy requests through MCP:

```shell
codex servers add ctx \
  --command /usr/local/bin/ctx \
  --args --mcp \
  --cwd /Users/alex/src/project
```

List registered servers with `codex servers list` and remove them with
`codex servers remove <name>`. Consult the Codex MCP documentation if your
installation uses a different configuration path.

Press `Ctrl+C` (or send `SIGTERM`) in the terminal that launched `ctx --mcp` to
shut the server down. The process waits up to five seconds for in-flight
requests to complete before exiting.

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

```shell
ctx tree . --tokens --summary
```

The summary line now reports total tokens (and the tokenizer model when applicable) alongside file counts and sizes, and each file entry includes its estimated token usage and model in JSON and XML output.

#### Testing Python helpers

To exercise the embedded Python helpers, install the required packages (for example, `pip install anthropic sentencepiece`) and run:

```shell
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

```shell
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

```shell
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

```shell
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

   ```shell
   git tag vX.Y.Z
   git push origin master
   git push origin vX.Y.Z
   ```

Tags that begin with `v` trigger the release workflow, which builds binaries and uses the matching changelog section as release notes.

## License

ctx is released under the [MIT License](MIT-LICENSE).
