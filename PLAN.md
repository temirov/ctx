# Plan

## CT-07 Callchain Support For Python And JavaScript
- Review the existing Go callchain traversal in `internal/commands/callchain.go` and wrap it in a `CallchainAnalyzer` interface keyed by language constants.
- Extend the callchain command plumbing to resolve the target language from the requested symbol or configuration and dispatch to analyzer implementations.
- Implement Python and JavaScript analyzers using static parsing (AST via `github.com/smacker/go-tree-sitter` or manual parsing) to build caller/callee relationships while sharing output assembly logic.
- Update documentation collection flow so non-Go analyzers can request documentation via the shared collector.
- Add table-driven tests covering Go, Python, and JavaScript callchain outputs alongside failure cases when a symbol is missing.

## CT-08 Unify `t` And `c` Commands
- Inspect `internal/commands` to identify duplication between tree and content commands and extract a shared `StreamOptions` structure to consolidate logic.
- Make the `t` alias reuse the content command path internally with a `IncludeContent` flag defaulting to false to preserve tree behavior.
- Ensure configuration precedence (flags > local config > global config) remains intact and update CLI help text accordingly.
- Add integration tests verifying that `ctx t` and `ctx c --content=false` share identical outputs across formats.

## CT-09 Document Clipboard Functionality
- Update `README.md` to describe the `--clipboard` flag, platform caveats, and configuration interaction.
- Mirror the documentation updates in `NOTES.md` under the maintenance checklist and mark CT-09 as complete when done.

## CT-10 Enhance Raw Tree Format
- Refactor the raw formatter in `internal/output` to render ASCII tree connectors (`|`, `├──`, `└──`).
- Preserve summary and documentation sections while updating affected tests or golden fixtures.
- Add new tests verifying raw tree rendering for nested directories and binary entries.

## CT-11 MCP Server Mode
- Introduce a new Cobra command flag `--mcp` (or separate subcommand) that starts an MCP server using a dedicated package under `internal/services/mcp`.
- Implement capability advertisement and request handling with clear interfaces for future extensions.
- Write integration-style tests that spin up the MCP server in-process and assert capability responses.
- Document usage in README and notes.

## CT-01 Embedded Script Portability
- Research current handling of Python/JS assets when compiling to a single binary and document embedding strategies (e.g., `embed` package) in `NOTES.md`.
- Note any blockers or required tooling updates, flagging follow-up work if needed.

## CT-02 Windows Compatibility Plan
- Audit OS-specific behavior (path separators, clipboard dependency, config locations) and draft a remediation plan in `NOTES.md`.
- Identify configuration or build steps needed for Windows support and capture them as actionable items.
