# Changelog

## [v0.0.13] - 2025-09-05

### Highlights

* **Repository structure improvements.** Exposed a root-level `main.go` for easier CLI builds, with shared Cobra setup
  for consistency. (#60 2686dae, c63e0af)
* **More flexible exclusion handling.** Multiple adhoc exclusion patterns are now supported in a single run. (#61
  fec03ce, 6aa9093)
* **Streamlined release flow.** Documented release steps for contributors and automated CI to run only on Go file
  changes. (#62 822d428, d9c9cd5, #63 7900db2, 30c2263)

### Features ✨

* Expose root-level `main.go` entry point and shared CLI configuration. (#60 2686dae, c63e0af)

### Improvements ⚙️

* Allow multiple exclusion patterns via `-e/--exclude`. (#61 fec03ce, 6aa9093)

### Docs 📚

* Add release preparation and publishing instructions. (#62 822d428, d9c9cd5)

### CI & Maintenance

* Run tests only when Go files change. (#63 7900db2, 30c2263)

**Upgrade notes:** No breaking changes. Builds, flags, and outputs remain compatible.

## [v0.0.12] - 2025-09-04

### Highlights

* **End-to-end MIME detection.** All files now get a detected MIME type; `tree` nodes include `mime` for file entries. (
  #57 e7ad015, #56 c587e9c, b58837c, b628a44)
* **CLI UX overhaul.** Custom usage template, improved help layout, and command aliases shown in help; help displays
  when no command is provided. (#55 a99dc40, a93b618, #47 aa8f792, #45 3de79fb, 60f0d10)
* **Ignore system hardening.** Refined path-based ignore logic, centralized ignore constants, clarified docs, and added
  `[binary]` section support in `.ignore`. (#54 14b5190, #51 ff3756a, #52 96c691c, #32 a6f07cb, 31d4603)
* **Stricter output validation.** Centralized output-format checks. (#35 e2b78f7, c83a8b4)
* **XML output format.** Added `xml` to supported output formats. (#20 da50618, 8f72db5)
* **Much broader test coverage.** New and expanded tests across `content/ctx`, ignore handling (including nested
  `.gitignore`), MIME detection, utilities, and output. (#53 1f7585b, #50 048a866, #48 0267170, #30 52fb374, #27
  7a98f75, 9203bbb, a117010, 6fe58e4, 73b7b01)

### Features ✨

* Detect MIME type for all files and surface MIME in `tree` file nodes. (#57 e7ad015, #56 c587e9c, b58837c, b628a44)
* Add `xml` to list of output formats. (#20 da50618, 8f72db5)

### Improvements ⚙️

* Support repeatable exclusion flag with glob pattern matching for tree and content commands.
* Improve help output and adopt a custom usage template. (#55 a99dc40, a93b618, #47 aa8f792)
* Show command aliases in help (Cobra help template override). (#45 3de79fb, c311f97)
* Refine path-based ignore handling and centralize ignore constants/messages. (#51 ff3756a, #41 ac2da74, 22bb964,
  a64d9cd, 59c829b)
* Extract common path flags and flag-registration helpers for consistency. (#44 cab1575, 101c207)
* Validate output formats via a single function. (#35 e2b78f7, c83a8b4)

### Fixes 🐛

* Show help when no command is provided (CLI regression fix). (#31 5493242, 60f0d10)

### Docs 📚

* Clarify ignore pattern handling and document related utilities. (#52 96c691c, 31d4603)
* Document call-chain depth flag and related behavior. (#29 ed3382f, f29acd5)
* Add GoDoc comment for `findGitDirectory`. (#34 2004ec2, ee5d754)
* Update README with new features; remove manual changelog fragment. (#25 541caeb, 47834ad)

### Tests ✅

* Add content command test for nested `.gitignore` and broader `ShouldIgnore` coverage. (#53 1f7585b, f6d517c, 1abe325)
* Add fixtures (incl. Google Sheets add-on) and extend `ctx_test.go`. (#50 048a866, a117010)
* Add tests for internal utils and output functions; call-chain depth tests; binary MIME tests. (#48 0267170, 6adc791,
  9203bbb, 73b7b01)

### Refactors 🧹

* Use descriptive loop/variable names across codebase. (#40 5d44468, #38 066afa0, f52116b, e73466f)
* Remove unused constants and legacy/binary patterns from tree handling. (#39 c0cf7cd, 24f6ce6, 77a27f0)
* Remove legacy `show-binary-content` directive (now superseded by `[binary]` in `.ignore`). (446504c)

### CI & Maintenance

* Run Go tests on pull requests. (#7 d8df33c, c0c0c95)
* License update. (c31641f)

**Upgrade notes:** No breaking changes expected in this release. Existing CLI flags and outputs remain compatible; the
new XML format and richer help/alias displays are additive.

## [v0.0.11] - 2025-04-20

### Features ✨

1. **Binary Section in `.ignore`**
    - `.ignore` files now support a `[binary]` section listing patterns whose binary contents are base64-encoded in
      output.
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