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

- Keep `resolveAndValidatePaths` for `tree`/`content`. No path validation needed for `callchain` argument itself
  (handled during analysis).

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

- **Modify `runTool`** to branch on `types.CommandCallChain`.
    - Call `commands.GetCallChainData`.
    - Render via `output.RenderCallChainJSON` or `output.RenderCallChainRaw`.
- Existing logic for `tree`/`content` remains unchanged.

### Step 6: Implement Output Rendering (`output/output.go`)

- **Create `RenderCallChainJSON`** and **`RenderCallChainRaw`**.
- Modify `runTool` to call the correct renderer.

### Step 7: Helper Functions (`utils.go`, `config.go`)

- Ensure ignore logic is correct. No changes needed beyond existing implementation.

---

## 4. Test Plan

*(Add integration tests for `callchain`, format flags, JSON parsing, raw output patterns, and error/warning conditions
as described above.)*

---

## 5. Summary

This PRD details the end‑to‑end design for `ctx`, now including the `callchain` command with embedded documentation
support, updated CLI flags, and comprehensive tests.
