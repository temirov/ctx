# Refactoring Plan (CT-402)

## Objectives

- Realign the codebase with the confident-programming rules in `POLICY.md` (no hidden globals, single validation at edges, cohesive types).
- Reduce duplication inside the CLI orchestration to improve readability and enable targeted testing.
- Identify high-risk gaps (missing tests, brittle error handling) and define concrete remediation steps.

## Proposed Workstreams

### 1. Command orchestration modularisation (`internal/cli/cli.go:338-755`)

**Findings**
- `createTreeCommand`, `createContentCommand`, and `createCallChainCommand` repeat nearly identical flag plumbing, copy-default resolution, and `runTool` invocation logic.
- `runTool` mixes collector construction, token initialisation, command dispatch, and clipboard writing in ~120 lines, and hard-codes `os.Stdout`/`os.Stderr` instead of `command.OutOrStdout()`.
- Current test coverage (`internal/cli/cli_stream_test.go`) hits only a subset of permutations and is difficult to extend because behaviour is not data-driven.

**Plan**
1. Introduce a `commandDescriptor` struct describing: command kind, default format, config section pointer, and render options.
2. Extract reusable builders for `copy` defaults, token setup, and clipboard dispatch so each subcommand configures a descriptor and defers to a shared executor.
3. Replace direct `os.Stdout`/`os.Stderr` usage with the Cobra-provided writers to honour parent command redirection and make tests deterministic.
4. Split `runTool` into: `buildExecutionContext` (collector + token counter) and `executeCommand` (dispatch), each returning narrow structs or interfaces.
5. Cover descriptors with table-driven tests asserting flag/config precedence, copy-only semantics, and invalid format handling.

**Policy alignment**: Removes duplicated edge logic, encourages narrow interfaces, and prepares for smart constructors around command options.

### 2. Eliminate hidden global registry (`internal/commands/callchain.go:21`)

**Findings**
- `defaultCallChainRegistry` is a package-level variable, violating the “no hidden globals for behaviour” rule. Tests cannot inject alternative analysers.

**Plan**
1. Promote a `callchain.Service` struct that owns a `Registry` instance and expose a constructor accepting analyser implementations.
2. Update CLI wiring to instantiate the service once per execution and inject it into `runCallChain` (or its replacement) rather than pulling from the global.
3. Adapt unit tests to construct services with stub analysers, improving coverage of error paths.

**Policy alignment**: Makes dependencies explicit and enforces composition over hidden state.

### 3. Configuration merge fidelity (`internal/config/app_config.go:170-203`)

**Findings**
- `StreamCommandConfiguration.merge` assigns `DocumentationMode` twice (lines 182-186), signalling duplication and risking missed invariants.
- The configuration structs are passed around as mutable bags; there are no smart constructors enforcing valid combinations (e.g., copy-only implies copy).

**Plan**
1. Remove the duplicate setter and add table-driven tests covering documentation mode precedence.
2. Introduce typed accessors (e.g., `TreeConfig.ToOptions()`) that collapse pointer fields into concrete command options with enforced invariants (`copyOnly => copy`, default format fallback, deduplicated excludes).
3. Ensure CLI commands call the new accessors, keeping validation at the configuration edge.
4. Extend `app_config_test.go` with cases that prove local overrides defeat global settings, pointer handling works, and copy/copy-only rules hold.

**Policy alignment**: Provides smart constructors for configuration-derived domain types and removes duplicated validation.

### 4. Stream pipeline dependency injection (`internal/commands/tree_stream.go`, `content_stream.go`)

**Findings**
- Warning paths write straight to `os.Stderr`, ignoring the injected warning handler (`TreeStreamOptions.Warn`) for some scenarios.
- `walkDirectory` aborts on `os.ReadDir` failure even after warning, preventing resilient traversal.
- Tree/content traversals duplicate binary handling and token counting fallbacks.

**Plan**
1. Replace direct `fmt.Fprintf(os.Stderr, …)` calls inside `StreamContent` with the supplied warning callback and enrich messages with stable codes.
2. Adjust `walkDirectory` to continue after non-fatal read errors when `Warn` is provided, returning partial summaries instead of aborting the command.
3. Extract shared helpers for determining binary output and token counting so tree/content stay in sync and share tests.
4. Add regression tests that simulate unreadable paths, token counter failures, and binary include/exclude toggles.

**Policy alignment**: Injects all side effects, keeps traversal pure aside from injected callbacks, and removes duplicated logic.

### 5. Remote documentation & token helpers (`internal/cli/cli.go:693-748`, `internal/tokenizer`)

**Findings**
- Remote documentation attempts are toggled inside `runTool` by inspecting documentation mode; the logic is hard-coded and untested beyond happy paths.
- `tokenOptions` silently ignores invalid model names until helper invocation; errors lack contextual wrapping.

**Plan**
1. Introduce a `DocumentationOptions` value object with explicit factory methods for disabled/relevant/full modes, capturing whether remote attempts are permitted.
2. Move GitHub token resolution into a dedicated edge function returning a typed result so failure cases wrap errors with `operation+subject` codes.
3. Expand tokenizer constructors to return sentinel errors when helpers are missing, and add tests covering fallback semantics and error propagation.
4. Cover combinations of `--doc`, `--docs-attempt`, and config defaults via new CLI integration tests ensuring remote fetchers are only activated when allowed.

**Policy alignment**: Strengthens invariants around documentation modes and improves error wrapping for helper initialisation.

### 6. Test coverage & safety nets

- Add end-to-end tests verifying `--copy-only` with `tree`/`content` respects stdout silence while clipboard receives data.
- Extend MCP integration tests to assert contextual error payloads for unsupported commands and invalid payloads.
- Backfill unit tests for `utils.RelativePathOrSelf` and ignore helpers to lock behaviour before refactors.

## Suggested Sequence

1. Land configuration fixes (Workstream 3) to stabilise option derivation.
2. Refactor command orchestration (Workstream 1) while injecting the call-chain service (Workstream 2).
3. Address stream pipeline injection and resilience (Workstream 4) to keep CLI behaviour stable during modularisation.
4. Harden documentation/token helpers (Workstream 5) and expand the safety-net tests (Workstream 6).

This ordering keeps validation and dependency injection improvements in place before touching high-level behaviour, aligning with the confident-programming policy.
