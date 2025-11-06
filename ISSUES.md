# ISSUES (Append-only Log)

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here. Read @NOTES.md for the process to follow when fixing issues.

Read @AGENTS.md, @ARCHITECTURE.md, @POLICY.md, @NOTES.md, @README.md and @ISSUES.md. Start working on open issues. Work autonomously and stack up the PRs.

## Features (100–199)

## Improvements (200–299)

- [x] [CT-02] add subcommands --copy-only, and add an abbreviated version of it, --co, which doesnt print the output of the command to STDOUT and only copies it to clipboard
    - Added persistent `--copy-only`/`--co` flags plus config support so clipboard-only mode skips stdout while preserving copy semantics across commands.
- [x] [CT-03] abbreviate --copy subcommand to --c, both staying valid and copying to the clipboard the output
    - Registered the `--c` shorthand for `--copy`, wired configuration detection to respect the alias, and documented the new toggle.
- [x] [CT-04] integrate toons with ctx and make it a default output format https://github.com/toon-format/toon
    - Default output is now TOON with new renderers and documentation updates covering the format change.


## BugFixes (300–399)

## Maintenance (400–499)

- [x] [CT-400] Update the documentation @README.md and focus on the usefullness to the user. Move the technical details to ARCHITECTURE.md
    - README now leads with user workflows and examples, while detailed architecture guidance lives in the new `ARCHITECTURE.md`.
- [x] [CT-401] Ensure architrecture matches the reality of code. Update @ARCHITECTURE.md when needed. Review the code and prepare a comprehensive ARCHITECTURE.md file with the overview of the app architecture, sufficient for understanding of a mid to senior software engineer.
    - Expanded `ARCHITECTURE.md` with package layout, subsystem responsibilities, configuration flow, and MCP/call-chain internals that mirror the current implementation.
- [x] [CT-402] Review @POLICY.md and verify what code areas need improvements and refactoring. Prepare a detailed plan of refactoring. Check for bugs, missing tests, poor coding practices, duplication and slop. Ensure strong encapsulation and following the principles og @AGENTS.md and policies of @POLICY.md
    - Added `REFACTORING_PLAN.md` outlining modularisation, configuration fixes, streaming resilience work, and coverage additions aligned with the confident-programming policy.

## Planning
do not work on the issues below, not ready
