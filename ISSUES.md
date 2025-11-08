# ISSUES (Append-only Log)

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here. Read @NOTES.md for the process to follow when fixing issues.

Read @AGENTS.md, @ARCHITECTURE.md, @POLICY.md, @NOTES.md, @README.md and @ISSUES.md. Start working on open issues. Work autonomously and stack up the PRs.

## Features (100–199)

- [x] [CT-100] Add a `discover` subcommand under doc command. The output of the command ctx doc discover will be a doc folder with the documentation of major dependencies in place.
Examples (generalize these)
1. If the frontend depends on beercss or bootsrtap 5 then documentation for these libraries will be extracted and saved under doc as md files, one per library (similarly to how --doc works now)
2. If the backend depends on gorm and viper then the documentation for these libraries will be extracted and saved under doc as md files, one per library (similarly to how --doc works now)

Plan the execution and consider the discovery phase (we can use an llm client or some other method to figure out where to search for documentation, starting with github). The deliverable is an implementation plan, not the code.
    - Documented the CLI, architecture, detection heuristics, and testing strategy for `ctx doc discover` in `docs/doc_discover_plan.md`, detailing Go/JS/Python discovery flows, repository/website/LLM resolvers, persistence layout, and rollout phases to unblock CT-101.
- [x] [CT-101] Implement the technical tasks delivered by CT-100. Consider integration tests with real data and ensure that the documentation for the 3rd party dependncies is pulled, processed and saved indeed
    - Implemented `ctx doc discover` with Go/npm/Python detectors, registry clients, Markdown writers, configuration support, clipboard integration, and fresh CLI/runner/unit/integration tests that verify real-world flows through mocked GitHub/npm/PyPI servers.

## Improvements (200–299)

- [x] [CT-02] add subcommands --copy-only, and add an abbreviated version of it, --co, which doesnt print the output of the command to STDOUT and only copies it to clipboard
    - Added persistent `--copy-only`/`--co` flags plus config support so clipboard-only mode skips stdout while preserving copy semantics across commands.
- [x] [CT-03] abbreviate --copy subcommand to --c, both staying valid and copying to the clipboard the output
    - Registered the `--c` shorthand for `--copy`, wired configuration detection to respect the alias, and documented the new toggle.
- [x] [CT-04] integrate toons with ctx and make it a default output format https://github.com/toon-format/toon
    - Default output is now TOON with new renderers and documentation updates covering the format change.
- [x] [CT-205] Modularise CLI command orchestration and execution pipeline
    - Introduced shared command descriptors, split execution into context-building and run phases with Cobra-managed writers, and expanded clipboard/format table tests to lock behaviour.
- [x] [CT-206] Inject callchain registry dependencies explicitly
    - Replaced the package-level registry with an injected service, wired through CLI and MCP descriptors, and added tests covering missing-service errors and stubbed analyzers.
- [x] [CT-207] Harden configuration merge invariants and smart constructors
    - Added typed clipboard settings with enforced copy-only invariants, removed duplicate documentation mode assignments, and broadened unit coverage for merge precedence scenarios.
- [x] [CT-208] Improve stream traversal resilience and warning injection
    - Routed tree and content streams through shared file inspection helpers, funneled warnings via injected callbacks, and added tests to confirm traversal continues on token counter failures.
- [x] [CT-209] Tighten remote documentation and token helper error handling
    - Added a shared `documentationOptions` value object plus an environment-backed GitHub token resolver so CLI, MCP, and `ctx doc` share normalization, API base handling, and contextual errors when tokens are missing.
    - Introduced tokenizer helper sentinel errors (`tokenizer.ErrHelperUnavailable`), surfaced them through the CLI with regression tests, and updated docs-attempt integration coverage to pin the new token requirement.
- [x] [CT-210] Expand integration and unit safety nets before refactors
    - Added a Cobra-driven `content --copy-only` test, new MCP negative-path tests (method validation and nested command paths), and regression coverage for size/time/mime helpers in `internal/utils` so safety nets exist before broader refactors.
- [x] [CT-211] Change the folder to save discovered documentation from `doc` to `docs`
    - Default `ctx doc discover` output now targets `docs/dependencies`, including the runner default, CLI help, configuration scaffolding, docs, and tests verifying the new path.


## BugFixes (300–399)

## Maintenance (400–499)

- [x] [CT-400] Update the documentation @README.md and focus on the usefullness to the user. Move the technical details to ARCHITECTURE.md
    - README now leads with user workflows and examples, while detailed architecture guidance lives in the new `ARCHITECTURE.md`.
- [x] [CT-401] Ensure architrecture matches the reality of code. Update @ARCHITECTURE.md when needed. Review the code and prepare a comprehensive ARCHITECTURE.md file with the overview of the app architecture, sufficient for understanding of a mid to senior software engineer.
    - Expanded `ARCHITECTURE.md` with package layout, subsystem responsibilities, configuration flow, and MCP/call-chain internals that mirror the current implementation.
- [x] [CT-402] Review @POLICY.md and verify what code areas need improvements and refactoring. Prepare a detailed plan of refactoring. Check for bugs, missing tests, poor coding practices, duplication and slop. Ensure strong encapsulation and following the principles og @AGENTS.md and policies of @POLICY.md
    - Converted the refactoring roadmap into tracked issues (CT-205 – CT-210) to align with the confident-programming policy and removed the standalone plan document.

## Planning
do not work on the issues below, not ready
- JS discovery improvements:
  - Allow recursive manifest detection (e.g., `**/package.json` outside `node_modules/`) so repos with nested web apps get documented in one pass. Needs glob/ignore support, caching, and CLI flags (`--package-json-glob`?) to keep large workspaces manageable. Must dedupe dependencies discovered in multiple subtrees and keep output directories stable.
  - Optional LLM/search-backed hinting phase for “no docs found” cases. Prompt model with README text + directory listing to suggest candidate doc paths or external doc URLs (validated before fetch). Requires provider flag (`--llm-provider`), caching responses per dependency, domain allowlist for external fetches, and clear user-facing opt-in because of latency/cost.
