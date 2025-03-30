# Product Requirements Document (PRD)

## 1. Overview

**Utility Name:** `content`

**Purpose:**
The `content` utility is a command‑line tool written in Go. It provides two primary functions for one or more specified
file and/or directory paths:

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
        - For directory paths: Prints a directory tree view.
        - For file paths: Prints the file's absolute path prefixed with `[File]`.
    - `content` (or `c`):
        - For directory paths: Outputs the concatenated contents of non-ignored files found recursively within.
        - For file paths: Outputs the content of the specified file directly.
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

- `main.go`: Entry point, argument parsing, path validation, orchestration logic.
- `config.go`: Loader for ignore files.
- `commands/`: Package containing command logic for **directory** processing.
    - `tree.go`: Implements recursive tree view for a directory.
    - `content.go`: Implements recursive content gathering for a directory.
- `utils.go`: Helper functions (ignore pattern matching, directory check).

### Step 1: Argument Parsing (`main.go`)

- **Parse Command‑Line Arguments:**
    - Identify the command (`tree` or `content`).
    - Capture all positional arguments *before* flags as `inputPaths []string`.
    - Default to `inputPaths = []string{"."}` if empty.
    - Parse global flags (`-e`/`--e`, `--no-gitignore`, `--no-ignore`).
- **Validation:** Ensure correct command, argument order (paths before flags), flag values. Display usage on error.

### Step 2: Path Validation (`main.go`)

- **Create `resolveAndValidatePaths`:**
    - Takes `inputPaths []string`.
    - Returns `[]ValidatedPath`, `error`. (`type ValidatedPath struct { AbsolutePath string; IsDir bool }`).
    - Iterates `inputPaths`:
        - Resolve absolute path (`filepath.Abs`).
        - Clean path (`filepath.Clean`).
        - Use a map on cleaned absolute paths to detect and skip duplicates.
        - Use `os.Stat` to check existence (error if not found) and determine `IsDir`.
        - Append `ValidatedPath` struct to results.
    - Return unique, validated paths with type information.

### Step 3: Orchestration (`main.go`)

- **Rename `runMultiDirectoryContentTool` to `runContentTool`:**
    - Call `resolveAndValidatePaths`. Handle error.
    - Iterate through the `[]ValidatedPath`.
    - **Switch on `commandName`:**
        - **`tree`:**
            - If `IsDir`: Load ignores for dir (using `loadIgnorePatternsForDirectory`), print header, call
              `commands.TreeCommand`. Handle errors.
            - If `!IsDir`: Print `[File] path`.
        - **`content`:**
            - If `IsDir`: Load ignores for dir (using `loadIgnorePatternsForDirectory`), call `commands.ContentCommand`.
              Handle errors.
            - If `!IsDir`: Call `printSingleFileContent(path)`. Handle errors (warnings).
    - Track the first error encountered for final exit status.

### Step 4: Implement File Content Printing (`main.go`)

- **Create `printSingleFileContent`:**
    - Takes `filePath string`. Returns `error` (though likely returns `nil` after printing warning).
    - Prints standard `File: ...` header.
    - Uses `os.ReadFile`.
    - On error: Prints warning to `stderr`, prints separator, returns `nil`.
    - On success: Prints content, prints `End of file: ...` footer, prints separator, returns `nil`.

### Step 5: Update Ignore Loading (`main.go`)

- **Modify `loadIgnorePatternsForDirectory`:** Ensure it's only called for directories. The logic to add the `EXCL:`
  prefix for the `-e` flag remains correct within this context.

### Step 6: Command Implementations (`commands/`)

- **`commands.TreeCommand`:** No changes needed (header moved to `main`).
- **`commands.ContentCommand`:** No changes needed. Works on directories passed to it.

### Step 7: Helper Functions (`utils.go`, `config.go`)

- No changes needed in `utils` or `config`.

---

## 4. Test Plan

*(Adding new test cases for mixed inputs and scoping)*

### 1. Argument Parsing and Validation

- (Existing cases remain valid for argument order and flag parsing)
- **Valid Input Case:** `content c file.txt dir1 -e log` (Expect: Command "content", Paths ["file.txt", "dir1"],
  Exclusion "log")

### 2. Path Validation

- **Case:** Mixed valid paths: `content c existing_file.txt existing_dir` (Expect: Proceeds, `resolveAndValidatePaths`
  returns list with correct `IsDir` flags).
- **Case:** Non-existent file: `content c non_existent_file.txt` (Expect: Error exit from `resolveAndValidatePaths`).
- **Case:** Mixed with non-existent: `content c valid_dir non_existent_file.txt` (Expect: Error exit).
- **Case:** Duplicate file/dir paths: `content c file.txt ./file.txt dir ./dir` (Expect: Processes `file.txt` once,
  `dir` once).

### 3. Configuration File (`.ignore`, `.gitignore`) Loading & Scoping

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

- **Case:** Mixed input tree: `content t fileA.txt dirB fileC.txt`. (Expect: `[File] .../fileA.txt`, then
  `--- Directory Tree: .../dirB ---` + tree, then `[File] .../fileC.txt`). Order matches arguments.

### 6. Content Command Functionality

- **Case:** Mixed input content: `content c fileA.txt dirB fileC.txt`. (Expect: Content of fileA, then content of files
  in dirB (respecting ignores), then content of fileC. Order matches arguments).
- **Case:** Unreadable explicit file: Create unreadable `unreadable.txt`. Run `content c readable.txt unreadable.txt`. (
  Expect: Content of `readable.txt` printed; warning about `unreadable.txt` on stderr; command finishes successfully).

### 7. Integration and Edge Cases

- (Existing cases remain relevant)

### Testing Approach

- **Unit Tests:** (Focus on argument parsing, `resolveAndValidatePaths`)
- **Integration Tests:**
    - Add tests specifically for `TestMixedInput_Tree`, `TestMixedInput_Content`.
    - Add tests for `TestMixedInput_IgnoreScope_Content`, `TestMixedInput_IgnoreScope_Tree`.
    - Add tests for `TestMixedInput_EFlagScope`.
    - Add tests for `TestInput_NonExistentFile`, `TestInput_UnreadableFile_Content`.
    - Ensure existing multi-dir tests still pass.

---

## 5. Summary

This PRD details the specifications for the `content` utility, now enhanced to handle mixed file and directory inputs:

- **Functionality:** Provides `tree` and `content` commands processing a list of specified file and/or directory paths.
- **Behavior:** `tree` lists files and shows directory structures. `content` shows file content directly or recursively
  scans directories.
- **Exclusions:** Ignore rules (`.ignore`, `.gitignore`, `-e`) apply **only** during directory traversal and do not
  filter explicitly listed files. Flags `--no-ignore` and `--no-gitignore` disable loading ignore files during directory
  scans.
- **Coding Plan:** Outlines changes primarily in `main.go` for argument parsing, path validation (
  `resolveAndValidatePaths`), and orchestration logic dispatching to commands or direct file printing (
  `printSingleFileContent`). Command packages remain focused on directory processing.
- **Test Plan:** Expanded with comprehensive tests covering mixed inputs, ignore scoping rules, and error handling for
  different path types.