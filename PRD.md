# Product Requirements Document (PRD)

## 1. Overview

**Utility Name:** `content`

**Purpose:**
The `content` utility is a command‑line tool written in Go. It provides two primary functions for one or more specified
file and/or directory paths, with configurable output formats:

- **tree:** Displays directory structures and lists specified files.
- **content:** Outputs the content of specified files and the contents of files within specified directories.

The utility filters files and directories *during directory traversal* based on exclusion patterns specified in
configuration files (`.ignore`, `.gitignore`) located within each processed directory, and an optional global exclusion
flag (`-e`/`--e`). Explicitly listed files are never filtered.

---

## 2. Functional Requirements

### Commands & Arguments

- **Primary Commands:**
    - `tree` (or `t`):
        - Processes specified paths. Output depends on `--format`.
    - `content` (or `c`):
        - Processes specified paths. Output depends on `--format`.
- **Positional Arguments:**
    - **Input Paths:**
        - Optional. One or more file or directory paths can be provided after the command.
        - If no paths are specified, defaults to the current directory (`"."`).
        - Duplicate paths (after resolving to absolute paths) are processed only once.
        - Order of processing matches the order of arguments.
- **Flags:**
    - **`-e` or `--e` flag:**
        - Specifies a single folder name to exclude globally **during directory traversal**.
        - **Behavior:**
            - If a folder with this name exists as a *direct child* of *any* specified **directory** argument, it is
              entirely excluded from the traversal of that specific directory.
            - Nested folders with the same name are *not* excluded by this flag.
            - This flag has **no effect** on explicitly listed **file** arguments.
            - Example: `content c file.log dir1 dir2 -e log` will process `file.log`, skip traversal of `dir1/log` and
              `dir2/log` (if they exist), but not skip `dir1/subdir/log`.
    - **`--no-gitignore` flag:**
        - Optional. Disables loading of `.gitignore` files when processing **directory** arguments.
    - **`--no-ignore` flag:**
        - Optional. Disables loading of `.ignore` files when processing **directory** arguments.

### Exclusion Patterns (During Directory Traversal ONLY)

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

- Non-existent paths provided as arguments cause immediate program termination with an error.
- Unreadable files provided explicitly as arguments to the `content` command result in a warning printed to `stderr`,
  but the program continues processing other paths.

---

## 3. Coding Plan

### Project Structure

- `main.go`: Entry point, argument parsing, path validation, orchestration, output rendering.
- `types/types.go`: Defines shared data structures (`ValidatedPath`, `FileOutput`, `TreeOutputNode`).
- `config.go`: Loader for ignore files.
- `commands/`: Package containing logic to **collect data** for directory processing.
    - `tree.go`: Implements `GetTreeData` returning `[]*TreeOutputNode`.
    - `content.go`: Implements `GetContentData` returning `[]FileOutput`.
- `utils.go`: Helper functions.

### Step 1: Argument Parsing (`main.go`)

- **Parse Command‑Line Arguments:** Add parsing for `--format` flag, validate its value (`raw` or `json`), store the
  result, default to `raw`.

### Step 2: Define Data Structures (`types/types.go`)

- Create `types` package.
- Define `ValidatedPath`, `FileOutput`, `TreeOutputNode` structs with necessary fields and JSON tags.

### Step 3: Path Validation (`main.go`)

- Modify `resolveAndValidatePaths` to use/return `types.ValidatedPath`.

### Step 4: Refactor Command Logic (Data Collection) (`commands/`)

- Modify `commands.ContentCommand` to `GetContentData`, returning `[]types.FileOutput, error`. Remove printing logic.
  Handle read errors by warning and skipping file.
- Modify `commands.TreeCommand` to `GetTreeData`, returning `[]*types.TreeOutputNode, error`. Implement recursive node
  building (`buildTreeNodes`). Remove printing logic. Handle read errors by warning and skipping subdirectory.

### Step 5: Orchestration & Data Aggregation (`main.go`)

- **Modify `runContentTool`:**
    - Accept `outputFormat` parameter.
    - Create `collectedResults []interface{}`.
    - Loop through `validatedPaths`.
    - Call appropriate `Get*Data` function or create nodes/file output directly for file paths.
    - Append results (*pointers* to structs where applicable) to `collectedResults`.
    - Handle and track non-fatal processing errors/warnings.

### Step 6: Implement Output Rendering (`main.go`)

- **Create `renderJsonOutput`:** Takes `collectedResults`, marshals using `json.MarshalIndent`, prints to `stdout`.
- **Create `renderRawOutput`:** Takes `commandName`, `collectedResults`. Uses type assertions to determine result type (
  `*types.FileOutput` or `*types.TreeOutputNode`). Calls `printRawTreeNode` recursively for directory nodes in `tree`
  mode. Prints file content/markers based on `commandName`.
- Modify `runContentTool` to call the correct rendering function based on `outputFormat`.

### Step 7: Helper Functions (`utils.go`, `config.go`, `main.go`)

- Add default ignores for `.ignore`, `.gitignore` in `utils`.
- Ensure `deduplicatePatterns` is used correctly in `main`.

---

## 4. Test Plan

*(Adding tests for formats)*

### 1. Argument Parsing and Validation

- (Existing cases)
- **Case:** `--format json` valid.
- **Case:** `--format raw` valid.
- **Case:** `--format` missing value (Error).
- **Case:** `--format invalid` (Error).
- **Case:** Default format (no flag) is `raw`.

### 2. Path Validation

- **Case:** Mixed valid paths: `content c existing_file.txt existing_dir` (Expect: Proceeds, `resolveAndValidatePaths`
  returns list with correct `IsDir` flags).
- **Case:** Non-existent file: `content c non_existent_file.txt` (Expect: Error exit from `resolveAndValidatePaths`).
- **Case:** Mixed with non-existent: `content c valid_dir non_existent_file.txt` (Expect: Error exit).
- **Case:** Duplicate file/dir paths: `content c file.txt ./file.txt dir ./dir` (Expect: Processes `file.txt` once,
  `dir` once).

### 3. Configuration File Loading & Scoping

- **Case:** Ignore scope for explicit file (Content): `dir/.ignore` has `*.log`. Run `content c dir/app.log dir`. (
  Expect: `dir/app.log` content *is* printed first because explicitly listed; other `.log` files *within* `dir` are
  *not* printed during `dir` traversal).
- **Case:** Ignore scope for explicit file (Tree): `dir/.ignore` has `ignored.log`. Run
  `content t dir/ignored.log dir`. (Expect: Output includes `[File] .../dir/ignored.log` line; Tree output for `dir`
  *omits* `ignored.log`).

### 4. Glob Matching and Exclusion Logic (Scoping)

- **Case:** `-e` flag scope: `dir/log` exists. Run `content c file.log dir -e log`. (Expect: Content of `file.log` is
  printed; traversal of `dir` excludes the `log` subdirectory).

### 5. Tree Command Functionality

### 5. Output Format Testing

- **Case:** `content content --format json`: Verify output is valid JSON matching the `[]FileOutput` schema. Check
  content, paths, type. Verify ignored/unreadable files are omitted.
- **Case:** `content tree --format json`: Verify output is valid JSON matching the `[]TreeOutputNode` schema. Check
  structure, paths, names, types. Verify ignored files/dirs are omitted from `children`.
- **Case:** `content content --format raw` / Default: Verify output matches the original raw text format.
- **Case:** `content tree --format raw` / Default: Verify output matches the original raw text format.
- **Case:** Mixed input with `--format json` (both commands): Verify structure and content are correct.
- **Case:** Empty result set with `--format json`: Verify output is `[]`.

### 6. Error/Warning Handling

- (Existing cases for non-existent paths)
- **Case:** Unreadable file with `--format json`: Verify command succeeds, warning on stderr, file omitted from JSON
  stdout.
- **Case:** Unreadable file with `--format raw`: Verify command succeeds, warning on stderr, file content section shows
  error/is omitted appropriately in raw stdout.
- **Case:** Unreadable directory during tree build (`--format json`): Verify command succeeds, warning on stderr,
  directory might appear in JSON but with `children: null` or omitted depending on implementation.
- **Case:** Unreadable directory during tree build (`--format raw`): Verify command succeeds, warning on stderr, raw
  tree output indicates issue or skips the branch.

### Testing Approach

- **Unit Tests:** (Argument parsing, potentially node building logic if complex).
- **Integration Tests:**
    - Add specific tests for `--format json` output validation using `encoding/json.Unmarshal`.
    - Add tests validating default (`raw`) output.
    - Add tests for invalid `--format` flag usage.
    - Ensure warning checks (`runCommandWithWarnings`) correctly separate stdout (for JSON) and stderr (for warnings).

---

## 5. Summary

This PRD details the specifications for the `content` utility, now enhanced to handle mixed file/directory inputs and
provide configurable output formats:

- **Functionality:** Provides `tree` and `content` commands processing files/directories. Supports `raw` (default) and
  `json` output formats via `--format` flag.
- **Behavior:** `tree` lists files/shows directory structures. `content` shows file content/scans directories. Output
  adapts to format.
- **Exclusions:** Ignore rules apply **only** during directory traversal. Explicit files are never ignored.
- **Coding Plan:** Outlines refactoring to separate data collection (`commands/`) from output rendering (`main.go`).
  Introduces `types` package and `--format` flag parsing. Implements `renderJsonOutput` and `renderRawOutput`. Updates
  ignore logic in `utils`.
- **Test Plan:** Expanded with tests for format flag usage, JSON output validation, raw output regression, and error
  handling across formats.