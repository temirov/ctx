# Product Requirements Document (PRD)

## 1. Overview

**Utility Name:** `ctx`

**Purpose:**
The `ctx` utility is a command‑line tool written in Go. It provides two primary functions for one or more specified
file and/or directory paths, with configurable output formats:

- **tree:** Displays directory structures and lists specified files.
- **content:** Outputs the content of specified files and the contents of files within specified directories.
- **callchain:** Analyzes the call graph for a specified function within the repository.

The utility filters files and directories *during directory traversal* (for `tree` and `content`) based on exclusion
patterns specified in configuration files (`.ignore`, `.gitignore`) located within each processed directory, and an
optional global exclusion flag (`-e`/`--e`). Explicitly listed files are never filtered.

---

## 2. Functional Requirements

### Commands & Arguments

- **Primary Commands:**
    - `tree` (or `t`):
        - Processes specified paths. Output depends on `--format`.
    - `content` (or `c`):
        - Processes specified paths. Output depends on `--format`.
    - `callchain` (or `cc`):
        - Analyzes call graph for a specified function. Output depends on `--format`.
- **Positional Arguments:**
    - **Input Paths / Arguments:**
        - **`tree`, `content`:** Optional. One or more file or directory paths can be provided after the command. If
          none
          are specified, defaults to the current directory (`"."`). Duplicate paths (after resolving) are processed only
          once. Order matters.
        - **`callchain`:** Required. Exactly one function identifier (fully qualified name or unique suffix) must be
          provided after the command.
- **Flags:**
    - **`-e` or `--e` flag:** (Applies to `tree`, `content`)
        - Specifies a single folder name to exclude globally **during directory traversal**.
        - **Behavior:**
            - If a folder with this name exists as a *direct child* of *any* specified **directory** argument, it is
              entirely excluded from the traversal of that specific directory.
            - Nested folders with the same name are *not* excluded by this flag.
            - This flag has **no effect** on explicitly listed **file** arguments.
            - Example: `ctx c file.log dir1 dir2 -e log` will process `file.log`, skip traversal of `dir1/log` and
              `dir2/log` (if they exist), but not skip `dir1/subdir/log`.
    - **`--no-gitignore` flag:** (Applies to `tree`, `content`)
        - Optional. Disables loading of `.gitignore` files when processing **directory** arguments.
    - **`--no-ignore` flag:** (Applies to `tree`, `content`)
        - Optional. Disables loading of `.ignore` files when processing **directory** arguments.
    - **`--format <raw|json>` flag:** (Applies to all commands)
        - Optional. Specifies the output format. Defaults to `raw`.
    - **`--version` flag:**
        - Displays the application version and exits immediately.

### Exclusion Patterns (During Directory Traversal ONLY - for `tree`, `content`)

- **Source of Exclusions:**
    - Exclusion rules apply *only* when recursively processing specified **directory** paths.
    - Rules are retrieved from:
        1. `.ignore` file located in the root of *each* processed directory (if `--no-ignore` is not used).
        2. `.gitignore` file located in the root of *each* processed directory (if `--no-gitignore` is not used).
        3. The global `-e`/`--e` flag (applies to direct children directories).
    - Patterns from `.ignore` and `.gitignore` within a specific directory are combined and deduplicated for processing
      the traversal of *that directory only*.
- **Scope:**
    - Ignore rules **do not** filter explicitly listed **file** arguments.
- **`.ignore` / `.gitignore` File Format:**
    - Uses standard Gitignore semantics (glob patterns). One pattern per line. Non-empty lines not beginning with `#`
      considered.

### Error Handling

- Invalid command or flag combinations cause immediate program termination with usage info.
- Non-existent paths provided as arguments for `tree`/`content` cause immediate program termination with an error.
- Unreadable files provided explicitly as arguments to the `content` command result in a warning printed to `stderr`,
  but the program continues processing other paths.
- Failure during `callchain` analysis (package loading, SSA build, function not found) results in program termination
  with an error.

---

## 3. Coding Plan

### Project Structure

- `main.go`: Entry point, argument parsing, path validation, orchestration, output rendering.
- `types/types.go`: Defines shared data structures (`ValidatedPath`, `FileOutput`, `TreeOutputNode`,
  `CallChainOutput`).
- `config.go`: Loader for ignore files.
- `commands/`: Package containing logic to **collect data** for directory/repository processing.
    - `tree.go`: Implements `GetTreeData` returning `[]*TreeOutputNode`.
    - `content.go`: Implements `GetContentData` returning `[]FileOutput`.
    - `callchain.go`: Implements `GetCallChainData` returning `*CallChainOutput`.
- `output/output.go`: Handles rendering data to JSON or Raw text.
- `utils/utils.go`: Helper functions (ignore matching, path utils, versioning).

### Step 1: Argument Parsing (`main.go`)

- Update `parseArgsOrExit` to handle `callchain` command and its single argument. Validate argument counts per command.
- Update `printUsage` to reflect all commands and arguments accurately.

### Step 2: Define Data Structures (`types/types.go`)

- Define `CallChainOutput` struct with necessary fields and JSON tags.

### Step 3: Path Validation (`main.go`)

- Keep `resolveAndValidatePaths` for `tree`/`content`. No path validation needed for `callchain` argument itself (
  handled
  during analysis).

### Step 4: Implement Command Logic (Data Collection) (`commands/`)

- Implement `commands.GetCallChainData`.
    - Load packages using `golang.org/x/tools/go/packages`.
    - Build SSA program using `golang.org/x/tools/go/ssa/ssautil`.
    - Build static call graph using `golang.org/x/tools/go/callgraph/static`.
    - Find the target function node.
    - Traverse graph edges to find callers and callees.
    - Extract source code for relevant functions using `go/ast` and `go/printer`.
    - Return `*types.CallChainOutput` or error.

### Step 5: Orchestration & Data Aggregation (`main.go`)

- **Modify `runTool` (previously `runContentTool`):**
    - Add a branch for `commandName == types.CommandCallChain`.
        - Call `commands.GetCallChainData`.
        - Handle potential errors from data collection.
        - Pass the result to the appropriate rendering function (`RenderCallChainJSON` or `RenderCallChainRaw`).
    - Existing logic for `tree`/`content` remains largely the same, collecting results into `collectedResults`.

### Step 6: Implement Output Rendering (`output/output.go`)

- **Create `RenderCallChainJSON`:** Marshals `*types.CallChainOutput` to indented JSON.
- **Create `RenderCallChainRaw`:** Formats `*types.CallChainOutput` to human-readable text.
- Modify `main.go` (`runTool`) to call the correct rendering function based on `commandName` and `outputFormat`.

### Step 7: Helper Functions (`utils.go`, `config.go`)

- Ensure ignore logic correctly handles path separators and patterns (already implemented).

---

## 4. Test Plan

*(Adding tests for formats and callchain)*

### 1. Argument Parsing and Validation

- (Existing cases for tree/content, updated to use `ctx`)
- **Case:** `ctx callchain funcName` valid.
- **Case:** `ctx callchain` (missing funcName) (Error).
- **Case:** `ctx callchain funcA funcB` (too many args) (Error).
- **Case:** `ctx tree --format json` valid.
- **Case:** `ctx content --format raw` valid.
- **Case:** `ctx callchain funcName --format json` valid.
- **Case:** `ctx --format` missing value (Error).
- **Case:** `ctx tree --format invalid` (Error).
- **Case:** Default format (no flag) is `raw` for all commands.
- **Case:** Invalid command `ctx invalidcmd` (Error).
- **Case:** Version flag `ctx --version` (Prints version, exits 0).
- **Case:** `ctx content dir1 --no-ignore dir2` (Positional arg after flag) (Error)
- **Case:** `ctx callchain --format json funcName` (Positional arg after flag for callchain is OK) (Valid parsing, may
  fail later on func lookup)

### 2. Path Validation (`tree`, `content`)

- **Case:** Mixed valid paths: `ctx c existing_file.txt existing_dir` (Expect: Proceeds).
- **Case:** Non-existent file: `ctx c non_existent_file.txt` (Expect: Error exit).
- **Case:** Mixed with non-existent: `ctx c valid_dir non_existent_file.txt` (Expect: Error exit).
- **Case:** Duplicate file/dir paths: `ctx c file.txt ./file.txt dir ./dir` (Expect: Processes once).

### 3. Configuration File Loading & Scoping (`tree`, `content`)

- **Case:** Ignore scope for explicit file (Content): `dir/.ignore` has `*.log`. Run `ctx c dir/app.log dir`. (
  Expect: `dir/app.log` content *is* printed first; other `.log` files *within* `dir` are *not* printed).
- **Case:** Ignore scope for explicit file (Tree): `dir/.ignore` has `ignored.log`. Run
  `ctx t dir/ignored.log dir`. (Expect: Output includes `[File] .../dir/ignored.log` line; Tree for `dir` *omits*
  `ignored.log`).

### 4. Glob Matching and Exclusion Logic (Scoping - `tree`, `content`)

- **Case:** `-e` flag scope: `dir/log` exists. Run `ctx c file.log dir -e log`. (Expect: Content of `file.log` printed;
  traversal of `dir` excludes `log`).

### 5. Output Format Testing (`tree`, `content`)

- **Case:** `ctx content --format json`: Verify valid JSON `[]FileOutput`. Check content, paths, type. Verify ignored
  files omitted.
- **Case:** `ctx tree --format json`: Verify valid JSON `[]TreeOutputNode`. Check structure, paths, names, types. Verify
  ignored omitted.
- **Case:** `ctx content --format raw` / Default: Verify original raw text format.
- **Case:** `ctx tree --format raw` / Default: Verify original raw text format.
- **Case:** Mixed input with `--format json` (both): Verify structure/content correct.
- **Case:** Empty result set with `--format json`: Verify output is `[]`.

### 6. Call Chain Command Functionality (`callchain`)

- **Case:** `ctx callchain validFuncName --format raw`: Verify output contains expected Target, Callers, Callees, and
  Functions sections. Check specific known caller/callee. Check source code snippet presence.
- **Case:** `ctx callchain validFuncName --format json`: Verify output is valid JSON matching `CallChainOutput` schema.
  Check `targetFunction`, `callers` array, `callees` array, and `functions` map contain expected values/keys and
  source code.
- **Case:** `ctx callchain nonExistentFunc`: Verify program exits with error indicating function not found.
- **Case:** `ctx callchain ambiguousSuffix`: Verify program exits with error or provides specific feedback if suffix
  matches multiple functions (error preferred).
- **Case:** Function with no callers/callees: Verify raw/JSON output correctly shows "(none)" or empty/null arrays.

### 7. Error/Warning Handling

- (Existing cases for non-existent paths for `tree`/`content`)
- **Case:** Unreadable file (`content`, `--format json`): Verify command succeeds, warning on stderr, file omitted from
  JSON stdout.
- **Case:** Unreadable file (`content`, `--format raw`): Verify command succeeds, warning on stderr, raw output
  indicates
  error for that file.
- **Case:** Unreadable directory (`tree`, `--format json`): Verify command succeeds, warning on stderr, dir might appear
  with `children: null` or be omitted.
- **Case:** Unreadable directory (`tree`, `--format raw`): Verify command succeeds, warning on stderr, raw tree output
  indicates issue or skips branch.
- **Case:** Error during package load (`callchain`): Verify fatal error exit.
- **Case:** Error during SSA build (`callchain`): Verify fatal error exit.

### Testing Approach

- **Unit Tests:** (Argument parsing, ignore pattern matching, maybe node building/call graph traversal helpers).
- **Integration Tests:**
    - Use `os/exec` to run the compiled `ctx` binary.
    - Create temporary directories with test file structures.
    - Test `tree`, `content`, and `callchain` commands using `ctx`.
    - Add specific tests for `--format json` output validation (`encoding/json.Unmarshal`).
    - Add tests validating default (`raw`) output against expected strings/patterns.
    - Test invalid flag usage and argument combinations expecting error exits.
    - Use `runCommandWithWarnings` helper to check stderr for expected warnings while validating stdout.
    - Run call chain tests against the tool's own source code using the `ctx` command.

---

## 5. Summary

This PRD details the specifications for the `ctx` utility, now enhanced to handle mixed file/directory inputs, perform
call chain analysis, and provide configurable output formats:

- **Functionality:** Provides `tree`, `content`, and `callchain` commands. Supports `raw` (default) and `json` output
  formats via `--format` flag.
- **Behavior:** `tree` lists files/shows directory structures. `content` shows file content/scans directories.
  `callchain` analyzes function relationships. Output adapts to format.
- **Exclusions:** Ignore rules apply **only** during directory traversal for `tree`/`content`. Explicit files are never
  ignored.
- **Coding Plan:** Outlines adding `callchain` logic, separating data collection (`commands/`) from output rendering
  (`output/output.go`), and updating argument parsing/orchestration (`main.go`).
- **Test Plan:** Expanded with tests for the `callchain` command, format flag usage, JSON output validation, raw output
  regression, and error handling across formats and commands, using `ctx` in examples.
