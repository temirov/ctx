# Plan

## CT-06 Documentation Extraction Enhancements

- internal/docs
  - Audit existing collectors and refactor into strategy registry keyed by language constants.
  - Implement Python docstring parsing using a robust approach with graceful degradation when interpreter support is unavailable.
  - Implement JavaScript documentation parsing for JSDoc-style comments with clearly documented limitations.
  - Add new interfaces ensuring extensibility and dependency injection for testing.

- internal/config
  - Extend configuration schema to include documentation language toggles if needed and ensure defaults integrate with new documentation strategies.

- internal/utils (or new helper package)
  - Centralize language constants and file extension maps for reuse across documentation and callchain tasks.

- tests/
  - Add table-driven tests validating documentation extraction for Go, Python, and JavaScript samples when `--doc` flag is supplied. Ensure tests cover absence of interpreter scenarios.

- NOTES.md
  - Update CT-06 status once completed with summary of decisions.

## CT-07 Callchain Support for Python and JavaScript

- internal/commands or new analyzer package
  - Abstract current Go callchain analyzer and register Python/JavaScript analyzers sharing a common interface.
  - For Python, research use of external tooling or custom static analysis; document external dependency requirements.
  - For JavaScript, implement parser based on established libraries or static heuristics; ensure behavior documented and tested.

- tests/
  - Provide integration tests that run callchain command against fixtures for Go, Python, and JavaScript verifying expected output structure.

- NOTES.md
  - Capture research findings and mark CT-07 complete.

## CT-08 Command Unification

- internal/cli/cli.go and related command implementations
  - Refactor `t` alias to delegate to content command with configurable content inclusion flag.
  - Ensure configuration precedence (flags > local > global) continues to work for unified command.

- tests/
  - Add command-level tests verifying alias behavior and content toggling.

## CT-09 Documentation Update

- README.md
  - Document clipboard functionality and any new commands or flags introduced by previous features.

- tests (if applicable)
  - Ensure no broken links or references.

- NOTES.md
  - Mark maintenance task complete after documentation changes.

## CT-10 Raw Format Tree Representation

- internal/output or formatting package
  - Modify raw formatter to render ASCII tree using `|` and `├──` characters while preserving existing metadata.

- tests/
  - Add golden tests covering tree structure output for simple directories.

## CT-11 MCP Server Mode

- cmd/server or new entrypoint under cmd
  - Implement MCP server startup flagged by `--mcp` integrating with existing command architecture.

- internal/server (new package)
  - Encapsulate MCP server logic with injectable dependencies for logging and configuration.

- tests/
  - Add integration test ensuring MCP mode responds to capability advertisement.

- README.md / NOTES.md
  - Document MCP usage and mark issue complete.

## CT-01 Portability of Embedded Scripts

- internal/docs and NOTES.md
  - Research embedding strategy for Python scripts when running from compiled binary; document approach and any blockers.

## CT-02 Windows Compatibility Plan

- NOTES.md
  - Detail required changes for Windows support, including path handling, clipboard dependencies, and interpreter availability.
