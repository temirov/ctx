# Architecture

This document captures the internal structure of `ctx`, the command-line tool for navigating projects, inspecting source
files, and analysing call chains. The README focuses on user-facing flows; use this guide when you need to understand
how the CLI is assembled or when you are extending the codebase.

## Runtime Overview

- `cmd/ctx` wires the Cobra root command and subcommands. Configuration loading is handled by Viper so flags, config
  files, and environment variables share a single source of truth.
- `internal/cli` builds the command graph, normalises flags (including the shared boolean parser), and owns clipboard
  execution, MCP server bootstrapping, and documentation fetch orchestration.
- `internal/commands` resolves filesystem input, applies ignore rules, and emits structured events for tree and content
  traversal. Call-chain analysis delegates to language-specific analysers in `internal/callchain`.
- `internal/services/stream` defines the neutral event protocol used by every renderer. `internal/output` provides Toon,
  raw, JSON, and XML renderers that subscribe to the same event stream.
- `internal/docs` and `internal/docs/githubdoc` contain the GitHub documentation fetchers that power `ctx doc` and the
  docs attempt heuristics.
- `internal/tokenizer` owns token counting backends and helper launching.

## Command Pipeline

1. Cobra resolves the command, merges config and flags, and constructs a request struct.
2. The CLI layer constructs a streaming session through `internal/services/stream`.
3. `internal/commands` walks the filesystem (tree/content) or runs call-chain assembly, emitting events as work completes.
4. The selected renderer (`internal/output`) consumes events and writes formatted output incrementally.
5. Post-processing hooks (clipboard integration, summaries, token totals) run after the renderer completes.

The pipeline is streaming end-to-end; directory traversal and content emission never buffer entire trees before producing
output. This design keeps large projects responsive and ensures JSON/XML/Toon/raw stay in sync.

## Documentation Retrieval

`ctx doc` consumes repository coordinates (`--path`, optionally `--ref`) and uses the GitHub Contents API to stream
directory entries. Cleanup rules (`--rules`) can remove files before rendering. The same fetcher powers the
`--docs-attempt` flag on `content` and `callchain`, where the CLI inspects import graphs, infers GitHub repositories, and
hydrates documentation alongside source output.

Documentation mode (`--doc relevant | full`) determines how much metadata to embed. `relevant` limits output to symbols
referenced locally; `full` adds GitHub documentation bundles retrieved by the fetcher.

## Token Counting Services

Token counting is optional and enabled with `--tokens`. When active, directory summaries accumulate file counts, byte
totals, and token estimates. `internal/tokenizer` chooses the backend from the model prefix:

| Prefix | Backend | Notes |
|--------|---------|-------|
| `gpt-`, `text-embedding`, `davinci`, `curie`, `babbage`, `ada`, `code-` | OpenAI encoders via `tiktoken-go`; falls back to `cl100k_base` when encodings are missing. |
| `claude-` | Anthropic helper launched with [`uv`](https://github.com/astral-sh/uv); requires `ANTHROPIC_API_KEY`. |
| `llama-` | SentencePiece helper launched with `uv`; downloads a compatible tokenizer model on demand. |
| other | Defaults to `cl100k_base`. |

Set `CTX_UV` to override the helper executable. Integration tests can exercise helpers with
`CTX_TEST_PYTHON=python3 go test -tags python_helpers ./internal/tokenizer`. Set `CTX_TEST_RUN_HELPERS=1` to enable the
optional helper suite and `CTX_TEST_UV` to point at a custom `uv` binary.

## Binary File Handling

- When `.ignore` has no `[binary]` section, binary files are omitted from raw output. JSON/XML still surface `mimeType`
  metadata.
- Add a `[binary]` section listing glob patterns to emit base64-encoded payloads. Matched files appear as `type:
  "binary"` with inline content.
- Explicitly listed files are never filtered by ignore rules; traversal filters apply only to recursive directory walks.

## Streaming Renderers

Streaming events cover directory entry/exit, file metadata, content chunks, summaries, warnings, and errors. Renderers
map events to their respective formats:

- **Raw:** Human-readable tree blocks (`[File] path`) and inline summaries.
- **JSON:** Newline-delimited JSON events matching the schema in `internal/services/stream/events.go`.
- **XML:** `<events><event …></event></events>` envelopes with XML-serialised events.
- **Toon:** Structured Toon documents with sections for metadata, files, summaries, and documentation.

Downstream tools can rebuild aggregate views from the same event feed without extra branching in the CLI or renderers.

Run the streaming regression suite with:

```bash
go test ./internal/services/stream ./internal/output ./internal/cli
```

These tests verify event ordering and renderer behaviour.

## MCP Server

The root `--mcp` flag launches an HTTP server that advertises capabilities at `/capabilities`, reports its working
directory via `/environment`, and executes commands through `/commands/<name>`. Responses are always JSON, mirroring the
equivalent CLI output. The server listens on `127.0.0.1:0` and prints the bound address. Shutdown waits up to five
seconds for in-flight requests.

## Clipboard Integration

`--copy` (alias `--c`) mirrors rendered output to the system clipboard after the renderer completes. `--copy-only` (alias
`--co`) skips writing to stdout while keeping clipboard behaviour—ideal for piping results into other tools. Clipboard
requests fail fast with contextual errors when the platform is unsupported. Persistent defaults live in `config.yaml`
under the relevant command.

## Configuration Loading

- `.ignore` and `.gitignore` files are honoured at every directory root during traversal. Patterns apply only to the
  directory where they are defined.
- The repeatable `-e/--e` flag excludes direct child directories by name.
- Explicit file arguments bypass ignore filters entirely.
- `config.yaml` consolidates per-command defaults (format, summary, copy, documentation mode, etc.) and merges with flag
  overrides via Viper.

## Release Workflow

Follow these steps to publish a tagged release:

1. Update `CHANGELOG.md` with a new version section.
2. Commit the changelog update.
3. Tag the commit and push both the branch and the tag:

   ```bash
   git tag vX.Y.Z
   git push origin master
   git push origin vX.Y.Z
   ```

Tags starting with `v` trigger the automated release workflow, build platform binaries, and extract release notes from
the matching changelog section.
