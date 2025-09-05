# Changelog

## [v0.0.11] - 2025-04-20

### Features ✨

1. **Binary Section in `.ignore`**
    - `.ignore` files now support a `[binary]` section listing patterns whose binary contents are base64-encoded in output.
    - The legacy `show-binary-content:` directive has been removed.

## [v0.0.10] - 2025-04-19

### Features ✨

1. **Documentation Extraction (`--doc` flag)**
    - Added the `--doc` boolean flag to the `content` (`c`) and `callchain` (`cc`) commands.
    - When enabled, ctx collects and embeds documentation for imported third‑party packages and referenced functions.
    - Documentation is available in both *raw* and *json* output formats.

### Improvements ⚙️

1. **Tests Package Renaming**
    - All integration tests are now consolidated under the `tests` package to simplify discovery.
2. **Table‑Driven Tests**
    - Introduced table‑based patterns for new test cases to improve readability and maintenance.

### Internal

1. **README Update**
    - Added details about the new `--doc` flag and updated flag tables and examples.

---

## [v0.0.9] - 2025-04-15

### What's New 🎉

1. JSON is a default output format

## [v0.0.8] - 2025-04-15

### What's New 🎉

1. Renamed `content` to `ctx`

## [v0.0.7] - 2025-04-13

### Features ✨

1. **Call Chain Analysis Command (`callchain`, `cc`):**
    - Added a new command `callchain` (alias `cc`) to analyze the call graph for a specified function within the
      repository.
    - Usage: `content callchain <function_name> [--format <raw|json>]`
    - The `<function_name>` argument should be a fully qualified name (e.g., `github.com/org/repo/pkg.MyFunction`) or a
      unique suffix.
    - **Raw Output (`--format raw` or default):** Displays the target function, its direct callers, and its direct
      callees in a human-readable format, followed by the source code of these functions.
    - **JSON Output (`--format json`):** Outputs a JSON object containing the `targetFunction`, arrays of `callers` and
      `callees` (fully qualified names), and a `functions` map where keys are function names and values are their source
      code strings.

## [v0.0.6] - 2025-04-08

### Features ✨

1. **Version Flag:** Added a `--version` flag to display the application version.

### Bug Fixes 🐛

1. **Version Verbiage:** Corrected the wording used when displaying the version information.

## [v0.0.5] - 2025-03-29

### Features ✨

1. **JSON Output Format:**
    - Added a `--format <type>` flag.
    - Supported types are `raw` (default) and `json`.
    - `--format json` outputs results in a structured JSON format suitable for machine processing (e.g., by AI tools).
    - `content content --format json`: Outputs an array of objects, each with `path`, `type` ("file"), and `content` for
      successfully read files.
    - `content tree --format json`: Outputs an array representing input paths. Files have `path` and `type`. Directories
      have `path`, `type`, and nested `children`.
    - The default `raw` format preserves the original human-readable text output.

## [v0.0.3] - 2025-03-29

### Features ✨

1. **Multi-Directory Support:**
   The tool now accepts multiple directory paths as positional arguments.
   `content <command> [dir1] [dir2] ... [flags]`
    - `.ignore` and `.gitignore` files are loaded relative to *each* specified directory.
    - The `-e`/`--e` exclusion flag applies to direct children within *any* of the specified directories.
    - Output for `tree` shows separate trees for each directory.
    - Output for `content` concatenates file contents from all specified directories sequentially.
    - Duplicate input directories (after resolving paths) are processed only once.
    - If no directories are specified, it defaults to the current directory (`.`).

2. **Mixed File/Directory Input:**
   The tool now accepts both file and directory paths as positional arguments.
   `content <command> [path1] [path2] ... [flags]`
    - For the `content` command:
        - Explicitly listed files have their content printed directly.
        - Directories are traversed recursively, printing content of non-ignored files within them.
    - For the `tree` command:
        - Directories are traversed, displaying their structure.
        - Explicitly listed files are shown with a `[File]` marker and their absolute path.
    - Ignore rules (`.ignore`, `.gitignore`, `-e`) only apply during the traversal of *directory* arguments. Explicitly
      listed files are *never* filtered by ignore rules.

## [v0.0.2] - 2025-03-23

### What's New 🎉

1. **Using `.ignore` Instead of `.ignore`:**
   The tool now looks for a file named **.ignore** for exclusion patterns.

2. **Processing `.gitignore` by Default:**
   The tool loads **.gitignore** by default if present. Use the `--no-gitignore` flag to disable this.

3. **Folder Exclusion Flags:**
   Both **-e** and **--e** flags now work for specifying an exclusion folder. The folder is excluded only when it
   appears as a direct child of the working directory.

4. **Disabling Ignore File Logic:**
    - **--no-gitignore:** Prevents the tool from reading the **.gitignore** file.
    - **--no-ignore:** Prevents the tool from reading the **.ignore** file.

5. **Command Abbreviations:**
   Short forms **t** for **tree** and **c** for **content** are now supported.