# Plan

- PLAN.md
  - Summarize sequential work for CT issues; update as tasks complete.

- internal/cli/cli.go
  - Introduce configuration loading via Viper honoring priority CLI > local config > global config for shared options (format, summary, tokens, doc, clipboard, init mode).
  - Add persistent flags and initialization flow for --config, --init, --force, --clipboard, and ensure tree alias reuses content logic with content toggle per CT-08.
  - Refactor command execution to inject clipboard copier interface for CT-03 and route tree alias through unified content command implementation.
  - Ensure command constructors validate config during PreRunE and propagate context for OS portability (CT-01, CT-02 planning outputs).

- internal/config
  - Define configuration struct with serialization/deserialization helpers, defaults, merge logic, and YAML read/write utilities for CT-04 and CT-05.
  - Implement path resolution helpers for local (~/.ctx) and project-level config discovery plus init command support with overwrite logic respecting --force.

- internal/output
  - Extend stream renderers to support tee-ing into clipboard buffers while maintaining existing stdout/stderr behavior.
  - Provide helpers to capture final output strings for clipboard copy without altering streaming semantics.

- internal/utils
  - Add constants for new flag names, config filenames, directory names, and clipboard error messages.
  - Introduce OS abstraction utilities for portability planning (CT-01/CT-02) such as runtime detection and script execution helpers.

- internal/docs
  - Extend Collector with language-aware strategy registry supporting Go, Python (PEP 257 docstrings via embedded python AST helper), and JavaScript (JSDoc comment extraction) when --doc is requested per CT-06.
  - Abstract documentation loaders behind interface for easier extension and testing; add caching and error reporting improvements.

- internal/commands
  - Generalize callchain functionality through pluggable analyzers so Python and JavaScript call graphs can be produced (CT-07) via delegated services.
  - Align tree/content command implementation to reuse unified pipeline that toggles content inclusion, matching CT-08.

- internal/services (new/expanded packages)
  - Add clipboard service encapsulating github.com/atotto/clipboard with injectable interface for testing (CT-03).
  - Provide language-specific analyzers for Python/JavaScript leveraging external interpreters while documenting requirements (CT-01/CT-02).

- tests/
  - Add or expand integration tests covering new flags (--clipboard, --config precedence, --init), documentation extraction for Go/Python/JS, and multi-language callchain support.
  - Ensure tests remain table-driven and validate behavior, not implementation; add fixtures for Python/JS sources.

- README.md / NOTES.md
  - Document new flags, configuration workflow, clipboard behavior, multi-language doc/callchain capabilities, and cross-platform considerations. Update NOTES issue checklist as tasks complete.

- Additional artifacts
  - Introduce fixture directories under tests for Python/JS samples supporting new features while avoiding duplication.
  - Provide internal design documentation (doc.go or package comments) as needed for new packages or refactors.
