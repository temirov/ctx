# Notes

## Role

You are a staff level full stack engineer. Your task is to **re-evaluate and refactor the CTX (context) repository** according to the coding standards already written in **AGENTS.md**.

## Context

- AGENTS.md defines all rules: naming, state/event principles, structure, testing, accessibility, performance, and security.
- The repo uses Alpine.js, CDN scripts only, no bundlers.
- Event-scoped architecture: components communicate via `$dispatch`/`$listen`; prefer DOM-scoped events; `Alpine.store` only for true shared domain state.
- The backend uses Go language ecosystem

## Your tasks

1. **Read AGENTS.md first** → treat it as the _authoritative style guide_.
2. **Scan the codebase** → identify violations (inline handlers, globals, duplicated strings, lack of constants, cross-component state leakage, etc.).
3. **Generate PLAN.md** → bullet list of problems and refactors needed, scoped by file. PLAN.md is a part of PR metadata. It's a transient document outlining the work on a given issue.
4. **Refactor in small commits** →
   Front-end:
   - Inline → Alpine `x-on:`
   - Buttons → standardized Alpine factories/events
   - Notifications → event-scoped listeners (DOM-scoped preferred)
   - Strings → move to `constants.js`
   - Utilities → extract into `/js/utils/`
   - Composition → normalize `/js/app.js` as Alpine composition root
     Backend:
   - Use "object-oreinted" stye of functions attached to structs
   - Prioritize data-driven solutions over imperative approach
   - Design and use shared components
5. **Tests** → Add/adjust Puppeteer tests for key flows (button → event → notification; cross-panel isolation). Prioritize end-2-end and integration tests.
6. **Docs** → Update README and MIGRATION.md with new event contracts, removed globals, and developer instructions.
7. **Timeouts**  Set a timer before running any CLI command, tests, build, git etc. If an operation takes unreasonably long without producing an output, abort it and consider a diffeernt approach. Prepend all CLI invocations with `timeout -k <N>s -s SIGKILL <N>s` command. Theis is MANDATORY for each and every CLI command.

## Output requirements

- Always follow AGENTS.md rules (do not restate them, do not invent new ones).
- Output a **PLAN.md** first, then refactor step-by-step.
- Only modify necessary files.
- Descriptive identifiers, no single-letter names.
- End with a short summary of changed files and new event contracts.

**Begin by reading AGENTS.md and generating PLAN.md now.**

## Rules of engagement

Review the NOTES.md. Make a plan for autonomously fixing every item under Features, BugFixes, Improvements, Maintenance. Ensure no regressions. Ensure adding tests. Lean into integration tests. Fix every issue. Document the changes.

Fix issues one by one, working sequentially.

1. Create a new git bracnh with descriptive name, for example `feature/LA-56-widget-defer` or `bugfix/LA-11-alpine-rehydration`. Use the taxonomy of issues as prefixes: improvement/, feature/, bugfix/, maintenace/, issue ID and a short descriptive. Respect the name limits.
2. Describe an issue through tests.
   2a. Ensure that the tests are comprehensive and failing to begin with.
   2b. Ensure AGENTS.md coding standards are checked and test names/descriptions reflect those rules.
3. Fix the issue
4. Rerun the tests
5. Repeat pp 2-4 untill the issue is fixed:
   5a. old and new comprehensive tests are passing
   5b. Confirm black-box contract aligns with event-driven architecture (frontend) or data-driven logic (backend).
   5c. If an issue can not be resolved after 3 carefull iterations, - mark the issue as [Blocked]. - document the reason for the bockage. - commit the changes into a separate branch called "blocked/<issue-id>". - work on the next issue from the divergence point of the previous issue.
6. Write a nice comprehensive commit message AFTER EACH issue is fixed and tested and covered with tests.
7. Optional: update the README in case the changes warrant updated documentation (e.g. have user-facing consequences)
8. Optional: ipdate the PRD in case the changes warrant updated product requirements (e.g. change product undestanding)
9. Optional: update the code examples in case the changes warrant updated code examples
10. Mark an issue as done ([X])in the NOTES.md after the issue is fixed: New and existing tests are passing without regressions
11. Commit and push the changes to the remote branch.
12. Repeat till all issues are fixed, and commits abd branches are stacked up (one starts from another).

Do not work on all issues at once. Work at one issue at a time sequntially.

Leave Features, BugFixes, Improvements, Maintenance sections empty when all fixes are implemented but don't delete the sections themselves.

## Issues

### Features

  - [X] [CT-03] Add an ability to copy the output into the cliboard. Consider specifics of different OSs. add a flag to copy to clipboard --cliboard.
    ````markdown
        The most practical way is to use an existing cross-platform library rather than shelling out to `pbcopy` (macOS) / `xclip` (Linux) / `clip` (Windows) yourself. The most widely used one is:

        ### 1. Use `github.com/atotto/clipboard`

        This package abstracts away the platform specifics:

        ```go
        package main

        import (
            "fmt"
            "log"

            "github.com/atotto/clipboard"
        )

        func main() {
            textToCopy := "Hello from Go!"
            err := clipboard.WriteAll(textToCopy)
            if err != nil {
                log.Fatal(err)
            }
            fmt.Println("Copied to clipboard:", textToCopy)

            pasted, err := clipboard.ReadAll()
            if err != nil {
                log.Fatal(err)
            }
            fmt.Println("Read from clipboard:", pasted)
        }
        ```

        * **macOS** → uses native APIs (`pbcopy`/`pbpaste`).
        * **Linux** → requires `xclip` or `xsel` installed in the environment.
        * **Windows** → uses Win32 API directly (no extra dependency).

        ---

        ### 2. Minimal OS-specific approach (if you don’t want external libs)

        You can directly call the system clipboard utilities:

        * **macOS**:

        ```go
        exec.Command("pbcopy").Stdin.Write([]byte(text))
        ```
        * **Linux**:

        ```go
        exec.Command("xclip", "-selection", "clipboard").Stdin.Write([]byte(text))
        ```
        * **Windows**:

        ```go
        exec.Command("clip").Stdin.Write([]byte(text))
        ```

        But this requires branching per OS and ensuring the utilities are present.
    ````

  - [X] [CT-04] Allow for a configuration file config.yaml either locally or under ~/.ctx. Define the defaults and read them in the following priority: CLI flags (P0) -> local config (P1) -> global config (P2)
  - [X] [CT-05] Allow generation of the configuration file (local and global) with --init <local|global>, local by default, --force to overwrite the existing config.yaml, otherwise if the file exists then the program exits with an error
  - [X] [CT-06] Consider retrieving Python and JS code documentation when --doc flag is passed and teh code is in either JS or Python. Research if there is a standardized model of Python and JS party packages' documentation that can be employed.
    - Implemented language-aware documentation extractors. Go analysis now reuses a shared collector map while Python docstrings (module/class/function/method) are parsed via indentation-aware heuristics and JavaScript leverages JSDoc comment pairing. When go.mod is unavailable the collector gracefully continues for non-Go languages. Falling back to rune counting avoids network failures while keeping tokenized flows operational.
  - [X] [CT-07] Consider supporting a callchain for Python and JS. Research if there are either Go or native callchain detection functionality
    - Added registry-driven call chain analyzers for Go, Python, and JavaScript using tree-sitter based parsing, depth-limited traversal, and graceful fallback when no Go module is present.
  - [X] [CT-11] Add an MCP server and advertise program capabilities. use --mcp flag to run the program as an MCP server
  - [X] [CT-13] Extend MCP mode so HTTP clients can execute tree/content/callchain with documentation support, including repositories outside the working directory.
  - [X] [CT-15] Ensure MCP callchain analyzers and releases run with CGO enabled across CI and packaging.
  - [X] [CT-16] Add an ability to retrieve package documentation from GitHub.
      2. Add integration tests for verification. Use the following targets for tests
          2a. [jspreadsheet documentation](https://github.com/jspreadsheet/ce/blob/master/docs/jspreadsheet/docs/editors.md)
          2b. [marked.js documentation](https://github.com/markedjs/marked/tree/master/docs)
          2c. [beer.css documentation](https://github.com/beercss/beercss/tree/main/docs)
      3. Ensure that the documentation was fully extracted
      4. Plan to integrate a new command doc or d for short to retrieve the documentation
      5. Plan to extennd --doc flag to incorporate <full|relevant> portion of github documentation
      6. Carefully orchestrate the implementation
        6a. Full test coverage
        6b. MCP advertisement
        6c. Readme documentation
      - Implemented `internal/docs/githubdoc` fetcher, new `doc` CLI command with GitHub integration, MCP wiring, and regression tests for jspreadsheet, marked, and beercss documentation flows.
  - [ ] [CT-18] Intelligent document retrieval. Add the best efforts for document retrieval from the github repos. Make educated guesses and develop a heuristics to identify if a package is on github and has documentation under `tree/master/docs`. Implement under --docs-attempt flag. Expected behaviour is to identify the versions of the 3rd party packages and look up their documentation on github and retrieve it intelligently. This is for in-code dependencies only. The implementation of the retrieval shall be shared between the `doc` flag and  `--docs-attempt`

### Improvements

  - [X] [CT-08] Unify internal implementatiopn of the `t` and `c` commands as their only difference is the content of the files on the backend. make t an internal alis to c command with the flag --content false
  - [X] [CT-10] Change raw format to include graphic represwentation, similar to trre command of | and ├── characters to demonstrate the tree structure

### Maintenance

  - [X] [CT-09] Document copy to clipboard functionality in the README.md
  - [X] [CT-12] Document MCP usage with examples
    - README now walks through starting `ctx --mcp`, checking the health endpoint, and querying `/capabilities` with `curl`.

### Portability

  - [X] [CT-01] Consider how would the compiled Go executable run python scripts? Do we need to embed and recreate them at the runtime? Reflect the considerations in the NOTES.md under the respective items
    - Tokenizer helpers for Anthropic and LLaMA models live under `internal/tokenizer/helpers` and are embedded via `//go:embed`.
    - At runtime `materializeHelperScripts` writes the embedded helpers to a temporary directory before invoking them with `uv run`.
    - Shipping a single binary therefore only requires bundling the Go executable; the Python code and shell entry points are reproduced automatically, while the environment must provide `uv` and its dependencies.
  - [X] [CT-02] Check OS specific assumptions. Can we run the compiled version of this app on Windows? Prepare a plan of what needs to change to be able to run it on Windows. Reflect the considerations in the NOTES.md under the respective items
    - Path handling already relies on `filepath` so tree traversal and config resolution adapt to Windows path separators.
    - Clipboard support uses `github.com/atotto/clipboard`, which wraps the Win32 API and does not require shell utilities.
    - Token counters depend on the `uv` executable; Windows builds must bundle or document `uv.exe` (or set `CTX_UV`) and ensure Python dependencies install through uv.
    - MCP server uses `signal.NotifyContext` for shutdown; confirm Windows builds observe `SIGINT`/`SIGTERM` and extend coverage with CI jobs on Windows to validate end-to-end flows.

## BugFixes
  - [X] [CT-17] Add human readeable help to doc command with an explanation of required and optional parameters
    - `ctx doc --help` now lists required owner/repo/path inputs alongside optional repo URL, reference, rules, documentation mode, and clipboard flags.
    - Failing to provide repository coordinates surfaces actionable guidance that points to the relevant flags and reminds users to consult the help output.
  ```shell
    17:23:49 tyemirov@Vadyms-MacBook-Pro:~/Development/temirov/ctx - [master] $ go run main.go doc
  Error: doc command requires repository coordinates. Provide --owner, --repo, and --path or supply --repo-url. Run "ctx doc --help" for complete flag help.
  {"level":"fatal","ts":1760833434.554803,"caller":"ctx/main.go:19","msg":"application execution failed","error":"doc command requires repository coordinates. Provide --owner, --repo, and --path or supply --repo-url. Run \"ctx doc --help\" for complete flag help.","stacktrace":"main.main\n\t/Users/tyemirov/Development/temirov/ctx/main.go:19\nruntime.main\n\t/usr/local/opt/go/libexec/src/runtime/proc.go:285"}
  exit status 1
  ```
