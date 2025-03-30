# Product Requirements Document (PRD)

## 1. Overview

**Utility Name:** `content`

**Purpose:**
The `content` utility is a command‑line tool written in Go. It provides two primary functions:

- **tree:** Displays a directory tree view for one or more specified directories.
- **content:** Outputs the concatenated contents of files within one or more specified directories.

The utility filters out files and directories based on exclusion patterns specified in configuration files (`.ignore`,
`.gitignore`) located within each processed directory, and an optional global exclusion flag (`-e`/`--e`).

---

## 2. Functional Requirements

### Commands & Arguments

- **Primary Commands:**
    - `tree` (or `t`):
        - Prints a directory tree view for each specified root directory.
    - `content` (or `c`):
        - Outputs the concatenated contents of files under each specified root directory.
- **Positional Arguments:**
    - **Root Directories:**
        - Optional. One or more directory paths can be provided after the command.
        - If no directories are specified, defaults to the current directory (`"."`).
        - Duplicate paths (after resolving to absolute paths) are processed only once.
- **Flags:**
    - **`-e` or `--e` flag:**
        - Specifies a single folder name to exclude globally.
        - **Behavior:**
            - For both `content` and `tree` commands:
                - If a folder with this name exists as a *direct child* of *any* of the specified root directories, it
                  is entirely excluded from processing for that specific root directory.
                - Nested folders with the same name (e.g., `root1/subdir/excluded_folder`) are *not* excluded by this
                  flag.
                - Example: `content content dir1 dir2 -e log` will skip processing `dir1/log` and `dir2/log` if they
                  exist, but not `dir1/subdir/log`.
    - **`--no-gitignore` flag:**
        - Optional. Disables loading of `.gitignore` files from *all* processed directories.
    - **`--no-ignore` flag:**
        - Optional. Disables loading of `.ignore` files from *all* processed directories.

### Exclusion Patterns

- **Source of Exclusions:**
    - No built‑in exclusion patterns exist in the code.
    - Exclusion rules are retrieved from:
        1. `.ignore` file located in the root of *each* processed directory (if `--no-ignore` is not used).
        2. `.gitignore` file located in the root of *each* processed directory (if `--no-gitignore` is not used).
        3. The global `-e`/`--e` flag.
    - Patterns from `.ignore` and `.gitignore` within a specific directory are combined and deduplicated for processing
      that directory.
- **`.ignore` / `.gitignore` File Format:**
    - Uses standard Gitignore semantics (glob patterns).
    - One pattern per line.
    - Non-empty lines not beginning with `#` are considered.
    - Patterns loaded from a file in `dir1` only affect the processing of `dir1`.

---

## 3. Coding Plan

### Project Structure

- `main.go`: Entry point, argument parsing, multi-directory orchestration.
- `config.go`: Loader for ignore files (`.ignore`, `.gitignore`).
- `commands/`: Package containing:
    - `tree.go`: Implementation of the tree command (handles single directory context).
    - `content.go`: Implementation of the content command (handles single directory context).
- `utils.go`: Helper functions (directory validation, ignore pattern matching).

### Step 1: Argument Parsing (`main.go`)

- **Parse Command‑Line Arguments:**
    - Identify the command (`tree` or `content`).
    - Capture all positional arguments appearing *before* any flags as potential directory paths. Store in `[]string`.
    - If no paths provided, default the list to `[]string{"."}`.
    - Parse and store global flags (`-e`/`--e`, `--no-gitignore`, `--no-ignore`).
- **Validation:**
    - Ensure correct command.
    - Ensure positional arguments (directories) appear before flags.
    - Handle missing flag values.
    - Display usage instructions on error.

### Step 2: Directory Validation and Deduplication (`main.go`)

- Iterate through the list of directory paths provided by the parser.
- For each path:
    - Resolve to an absolute path (`filepath.Abs`).
    - Clean the path (`filepath.Clean`).
    - Use a map to track unique absolute paths and skip duplicates.
    - Validate that the path exists and is a directory (`utils.IsDirectory`). Return an error immediately if validation
      fails for any path.
- Store the list of unique, validated, absolute directory paths.

### Step 3: Multi-Directory Orchestration (`main.go`)

- Iterate through the list of validated, unique, absolute directory paths.
- For *each* directory:
    - **Load Exclusion Patterns:**
        - Initialize an empty pattern slice for this directory.
        - If `useIgnoreFile` is true, attempt to load `.ignore` from the current directory's absolute path using
          `config.LoadContentIgnore`. Append patterns. Handle `os.IsNotExist` gracefully.
        - If `useGitignore` is true, attempt to load `.gitignore` similarly. Append patterns. Handle `os.IsNotExist`
          gracefully.
        - Deduplicate the collected patterns for this directory.
        - If the global `exclusionFolder` flag is set, append the special `EXCL:<folder>` pattern.
    - **Execute Command:**
        - Call the appropriate command function (`commands.TreeCommand` or `commands.ContentCommand`) passing the
          current absolute directory path and its specific, calculated ignore patterns.
        - Handle errors returned by the command function (e.g., log a warning and continue with the next directory).

### Step 4: Implement Tree Command (`commands/tree.go`)

- **Modify `TreeCommand`:** Add a header print statement (`fmt.Printf`) before the tree logic to indicate which
  directory is being processed (e.g., `--- Directory Tree: /path/to/dir ---`).
- **Core Logic (`printTree`):** Remains largely unchanged. It receives a root path and ignore patterns and works
  recursively within that context. The `isRoot` logic for the `-e` flag check functions correctly based on the initial
  call for that specific directory.

### Step 5: Implement Content Command (`commands/content.go`)

- **Core Logic:** Remains unchanged. `ContentCommand` receives a root path and its specific ignore patterns.
  `filepath.WalkDir` traverses that path, and `handleContentWalkEntry` applies filters based on the provided patterns
  relative to that specific root. The output format naturally concatenates when the command is called multiple times by
  the orchestrator.

### Step 6: Helper Functions and Utilities (`utils.go`, `config.go`, `main.go`)

- **`utils.IsDirectory`:** No changes needed.
- **`utils.ShouldIgnore`, `utils.ShouldIgnoreByPath`:** No changes needed. They operate correctly based on the patterns
  and relative paths provided for the current context.
- **`config.LoadContentIgnore`:** No changes needed. It loads a specific file.
- **`main.deduplicatePatterns`:** Helper to deduplicate ignore patterns loaded for each directory.

---

## 4. Test Plan

*(Update existing sections and add new ones for multi-directory)*

### 1. Argument Parsing and Validation

- **Valid Input Cases:**
    - `content tree dir1 dir2` (Expect: Command "tree", Dirs ["dir1", "dir2"], no exclusion)
    - `content c dir1 -e log dir2` (Expect: Error, positional arg after flag) -> **Update**: This is now invalid. Expect
      `content c dir1 dir2 -e log`.
    - `content t dir1 --no-ignore dir2 -e log` (Expect: Error) -> **Update**: Invalid. Expect
      `content t dir1 dir2 --no-ignore -e log`.
    - `content c` (Expect: Command "content", Dirs ["."], no exclusion)
    - `content t .` (Expect: Command "tree", Dirs ["."], no exclusion)
    - `content content dir1 ./dir1` (Expect: Command "content", Dirs ["dir1", "./dir1"], no exclusion initially;
      deduplication happens later)
- **Error/Invalid Cases:**
    - `content tree dir1 extra -e log` (Expect: Error, positional arg after flag)
    - `content c -e` (Expect: Error, missing flag value)
    - `content invalid dir1` (Expect: Error, invalid command)
    - `content tree dir1 --unknown-flag` (Expect: Error, unknown flag)

### 2. Directory Validation

- **Case:** Existing directories provided: `content c existing_dir1 existing_dir2` (Expect: Proceeds)
- **Case:** One non-existent directory: `content c existing_dir1 non_existent_dir` (Expect: Exits with error about
  `non_existent_dir`)
- **Case:** One path is a file: `content c existing_dir1 path_to_file` (Expect: Exits with error about `path_to_file`)
- **Case:** Duplicate directories: `content c dir1 dir1` or `content c dir1 ./dir1` (Expect: Processes `dir1` only once)

### 3. Configuration File (`.ignore`, `.gitignore`) Loading

- **Case:** Specificity - `dir1/.ignore` has `*.log`, `dir2/.ignore` has `*.tmp`. Call `content c dir1 dir2`. (Expect:
  `.log` files ignored *only* in `dir1`, `.tmp` files ignored *only* in `dir2`).
- **Case:** Flags - `dir1` has `.ignore` and `.gitignore`. Call `content c dir1 --no-ignore`. (Expect: Only `.gitignore`
  patterns from `dir1` are applied). Call `content c dir1 --no-gitignore`. (Expect: Only `.ignore` patterns are
  applied). Call `content c dir1 --no-ignore --no-gitignore`. (Expect: No file patterns applied).

### 4. Glob Matching and Exclusion Logic

- **Case:** `-e` flag with multiple dirs - `dir1/log` exists, `dir2/log` exists, `dir1/subdir/log` exists. Call
  `content c dir1 dir2 -e log`. (Expect: `dir1/log` and `dir2/log` are excluded, `dir1/subdir/log` is included).
- **Case:** Overlapping patterns - `dir1/.ignore` has `*.txt`, `-e temp` is used. `dir1/temp/file.txt` exists. (Expect:
  `dir1/temp` is excluded due to `-e`, overriding the include otherwise implied for `file.txt`).

### 5. Tree Command Functionality

- **Case:** Multi-dir tree output - Call `content t dir1 dir2`. (Expect: Output shows a header
  `--- Directory Tree: <abs_path_dir1> ---` followed by its tree, then `--- Directory Tree: <abs_path_dir2> ---`
  followed by its tree. Ignores specific to each directory are respected).

### 6. Content Command Functionality

- **Case:** Multi-dir content output - Call `content c dir1 dir2`. (Expect: Concatenated output of files from `dir1` (
  respecting `dir1`'s ignores) followed by files from `dir2` (respecting `dir2`'s ignores)).

### 7. Integration and Edge Cases

- **Case:** Empty directory processing - `content c empty_dir other_dir`. (Expect: `empty_dir` produces no output,
  `other_dir` processed normally).
- **Case:** No files match in any dir - `content c dir_with_all_ignored`. (Expect: No file content output, completes
  without error).
- **Case:** `-e` flag matches dir in one root but not another - `dir1/log` exists, `dir2/data` exists. Call
  `content c dir1 dir2 -e log`. (Expect: `dir1/log` excluded, `dir2/data` included).

### Testing Approach

- **Unit Tests:** (Existing + potentially test argument parsing variations)
- **Integration Tests:**
    - Set up multiple temporary directories with distinct structures and ignore files.
    - Test `tree` and `content` commands with multiple directory arguments.
    - Verify ignore file specificity (`dir1/.ignore` vs `dir2/.ignore`).
    - Verify flag interactions (`--no-ignore`, `--no-gitignore`, `-e`) in multi-directory scenarios.
    - Test directory validation and deduplication logic.
    - Capture and assert output correctness for both commands.

---

## 5. Summary

This PRD details the specifications for the `content` utility:

- **Functionality:** Provides `tree` and `content` commands capable of processing one or more specified directories.
- **Exclusions:** Defined via `.ignore` and `.gitignore` files scoped to each processed directory, plus a global `-e`/
  `--e` flag for excluding direct child folders. Flags `--no-ignore` and `--no-gitignore` control file loading globally.
- **Coding Plan:** Outlines argument parsing for multiple directories, validation/deduplication, an orchestration loop
  in `main.go`, and minor adjustments to command outputs while keeping core command logic focused on single-directory
  contexts.
- **Test Plan:** Updated with comprehensive unit and integration tests covering multi-directory use cases, ignore
  scoping, flag interactions, and edge cases.