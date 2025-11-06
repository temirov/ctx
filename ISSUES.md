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

## Planning
do not work on the issues below, not ready
