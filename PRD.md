# Product Requirements Document (PRD)

## 1. Overview

**Utility Name:** `content`  
**Purpose:**  
The `content` utility is a command‑line tool written in Go. It provides two primary functions:

- **tree:** Displays a directory tree view.
- **content:** Outputs the contents of files within a directory.

The utility filters out files and directories based solely on exclusion patterns specified in a configuration file (
`.contentignore`) and an optional additional exclusion provided via the `-e` flag.

---

## 2. Functional Requirements

### Commands & Arguments

- **Primary Commands:**
    - `tree`:
        - Prints a directory tree view.
    - `content`:
        - Outputs the contents of files under a specified directory.
- **Positional Arguments:**
    - **Root Directory:**
        - Optional. Defaults to the current directory (`"."`) if not provided.
- **Flags:**
    - **`-e` flag:**
        - Specifies a single additional folder (relative to the working directory) to exclude.
        - **Behavior:**
            - For both the `content` and `tree` commands:
                - When the exclusion folder is located directly under the working directory (i.e. the root provided), it
                  is entirely excluded.
                - For example:
                    - `content.sh content pkg -e log` will skip processing all files under `pkg/log`.
                    - `content.sh tree pkg -e log` will exclude the `pkg/log` folder and its entire structure.
                    - `content.sh tree -e log` will exclude the `log` folder in the current directory (while nested
                      instances of a folder named `log` in subdirectories are not excluded).

### Exclusion Patterns

- **Source of Exclusions:**
    - No built‑in exclusion patterns exist in the code.
    - All exclusion rules are solely retrieved from a configuration file named `.contentignore` located in the current
      working directory.
- **`.contentignore` File Format:**
    - Uses full Git‑ignore semantic with one glob pattern per line.
    - Only non-empty lines and lines not beginning with `#` are considered.

---

## 3. Coding Plan

### Project Structure

- The implementation shall be organized into multiple files to promote modularity and maintainability.
- Suggested structure:
    - `main.go`: Entry point and argument parsing.
    - `config.go`: Loader for the `.contentignore` file.
    - `commands/`: A package containing:
        - `tree.go`: Implementation of the tree command.
        - `content.go`: Implementation of the content command.
    - `utils.go`: Helper functions such as directory validation and glob matching using full Git‑ignore semantics.

### Step 1: Argument Parsing

- **Parse Command‑Line Arguments:**
    - Identify the command (`tree` or `content`).
    - Capture the optional root directory; default to `"."` if not provided.
    - Parse and validate the optional `-e` flag and its folder name.
- **Validation:**
    - Ensure the correct number and order of arguments.
    - Display usage instructions on error.

### Step 2: Validate the Root Directory

- Use functions like `os.Stat` to verify that the specified directory exists and is accessible.
- Return an error if the directory is invalid.

### Step 3: Load Exclusion Patterns from `.contentignore`

- **File Check:**
    - Look for `.contentignore` in the current working directory.
- **Reading and Parsing:**
    - If the file exists, read it line by line.
    - Skip empty lines and comments (lines beginning with `#`).
    - Store valid glob patterns in a slice using full Git‑ignore semantics.
- **Integration:**
    - The collected glob patterns serve as the sole exclusion rules, which are merged with the additional exclusion
      provided by the `-e` flag.

### Step 4: Implement the Tree Command

- **Directory Traversal:**
    - Use recursive methods (e.g., `filepath.WalkDir`) to traverse the directory structure.
- **Output Formatting:**
    - Generate a tree‑like view with proper indentation.
- **Exclusion Logic:**
    - For each directory or file:
        - Apply full Git‑ignore semantics for exclusion using the patterns from `.contentignore`.
        - Apply the `-e` flag logic:
            - Exclude the folder only if it is a direct child of the working directory (root provided) that matches the
              exclusion folder.

### Step 5: Implement the Content Command

- **File Traversal:**
    - Recursively traverse the directory to find all files.
- **Filtering:**
    - Check each file’s path against the glob patterns from `.contentignore` using full Git‑ignore semantics.
    - Additionally, if `-e` is provided, skip files under the `<root>/<exclusionFolder>` directory when that folder is
      in the working directory.
- **Output:**
    - For each file that passes the filters:
        - Print the file path.
        - Print the file’s contents.
        - Output a separator (e.g., a dashed line) after each file.

### Step 6: Helper Functions and Utilities

- **Argument Parsing Function:**
    - Handle extraction and validation of the command, directory, and `-e` flag.
- **Directory Validation Function:**
    - Confirm that the provided root directory exists and is accessible.
- **Configuration Loader:**
    - Read and parse the `.contentignore` file to return a slice of ignore glob patterns.
- **Matching Functions:**
    - Implement glob matching using full Git‑ignore semantics to decide if a file or folder should be excluded.
- **Output Functions:**
    - Write separate functions to generate the tree view and output file contents, applying the exclusion logic.

---

## 4. Test Plan

### 1. Argument Parsing and Validation

- **Valid Input Cases:**
    - **Case 1:** `content tree pkg`
        - **Expectation:** Command parsed as `"tree"`, directory set to `"pkg"`, no additional exclusion.
    - **Case 2:** `content content pkg`
        - **Expectation:** Command parsed as `"content"`, directory set to `"pkg"`, no additional exclusion.
    - **Case 3:** `content tree pkg -e log`
        - **Expectation:** Command `"tree"`, directory `"pkg"`, exclusion flag set to `"log"`.
    - **Case 4:** `content content pkg -e log`
        - **Expectation:** Command `"content"`, directory `"pkg"`, exclusion flag set to `"log"`.
    - **Case 5:** `content tree -e log`
        - **Expectation:** Command `"tree"`, directory defaults to `"."`, exclusion flag set to `"log"`.

- **Error/Invalid Cases:**
    - **Case 6:** Too many arguments (e.g., `content tree pkg extra -e log`)
        - **Expectation:** Utility displays usage instructions or an error.
    - **Case 7:** Wrong flag order or missing flag value (e.g., `content tree pkg -e`)
        - **Expectation:** Returns an error or displays usage information.
    - **Case 8:** Invalid command (e.g., `content invalid pkg`)
        - **Expectation:** Displays usage message indicating valid commands.

### 2. Directory Validation

- **Case 9:** Existing directory provided
    - **Expectation:** Utility proceeds without error.
- **Case 10:** Non-existent directory provided
    - **Expectation:** Exits with an error message.
- **Case 11:** Provided path is not a directory (e.g., a file)
    - **Expectation:** Exits with an error indicating the path is not a directory.

### 3. Configuration File (`.contentignore`) Loading

- **Case 12:** `.contentignore` exists with valid patterns
    - **Test Data:**
      ```
      # Ignore log directories
      log/
      *.tmp
      ```
    - **Expectation:** Loader returns slice: `["log/", "*.tmp"]` (ignoring comment and blank lines).
- **Case 13:** `.contentignore` does not exist
    - **Expectation:** Loader returns an empty slice (or nil) so that only the `-e` flag exclusion applies if provided.
- **Case 14:** `.contentignore` contains only comments or blank lines
    - **Expectation:** Loader returns an empty slice.
- **Case 15:** `.contentignore` contains malformed lines (if applicable)
    - **Expectation:** Malformed lines are skipped or handled gracefully without crashing.

### 4. Glob Matching and Exclusion Logic

- **Case 16:** Directory glob matching using full Git‑ignore semantics
    - **Example:** Given a pattern `log/` in `.contentignore`, ensure that a directory path like `pkg/log` (with root
      `"pkg"`) is correctly excluded when `-e log` is specified.
- **Case 17:** File glob matching using full Git‑ignore semantics
    - **Example:** Given a pattern `*.tmp`, verify that files such as `file.tmp` or `pkg/dir/file.tmp` are excluded.
- **Case 18:** Additional `-e` flag verification
    - **Expectation:** The folder specified by `-e` is excluded only when it is directly under the working directory (
      either the current directory or the specified root).

### 5. Tree Command Functionality

- **Case 19:** Tree output without exclusions
    - **Expectation:** Output displays all files and folders in a tree‑like format.
- **Case 20:** Tree output with `.contentignore` patterns applied
    - **Test Data:**
        - Create a temporary directory structure with folders matching patterns in `.contentignore` (e.g., a top‑level
          folder `log`).
    - **Expectation:** Output omits directories/files matching the ignore patterns.
- **Case 21:** Tree command with additional `-e` flag
    - **Expectation:** When the exclusion folder is directly under the working directory, the tree output omits that
      folder and its entire structure regardless of whether the root is the current directory or a specified folder.

### 6. Content Command Functionality

- **Case 22:** File content output without exclusions
    - **Expectation:** All files in the directory tree are output along with their contents and a separator.
- **Case 23:** Content output with `.contentignore` applied
    - **Test Data:**
        - Create a temporary directory with files under a folder that matches an ignore pattern in `.contentignore`.
    - **Expectation:** Files matching the patterns are not output.
- **Case 24:** Content command with additional `-e` flag
    - **Expectation:** Files under `<root>/<exclusionFolder>` (where the exclusion folder is directly under the working
      directory) are skipped.

### 7. Integration and Edge Cases

- **Case 25:** Empty directory
    - **Expectation:** Utility handles an empty directory gracefully without errors.
- **Case 26:** Directory structure where no files pass the exclusion filters
    - **Expectation:** Utility completes execution, possibly with a message indicating no files/folders processed.
- **Case 27:** Complex directory structure with overlapping glob patterns and `-e` flag
    - **Expectation:** Combined exclusion logic correctly filters out intended directories and files.
- **Case 28:** Running in a working directory without a `.contentignore` file, using `-e`
    - **Expectation:** Only the `-e` flag exclusion is applied.

### Testing Approach

- **Unit Tests:**
    - Utilize Go’s `testing` package to write tests for:
        - Argument parsing.
        - Directory validation.
        - `.contentignore` loader.
        - Glob matching and exclusion functions.
        - Tree output and file content output functions.
- **Integration Tests:**
    - Create temporary directories/files using functions such as `os.MkdirTemp` or similar.
    - Capture and verify output against expected results.
- **Edge Cases:**
    - Include tests for empty directories, malformed `.contentignore` files, and overlapping ignore patterns.

---

## 5. Summary

This PRD details the specifications for the `content` utility:

- **Functionality:** Provides `tree` and `content` commands with exclusion capabilities.
- **Exclusions:** Are defined solely via a `.contentignore` configuration file (using full Git‑ignore semantics) and an
  optional `-e` flag, which excludes a folder only when it appears directly under the working directory.
- **Coding Plan:** Outlines argument parsing, directory validation, configuration file loading, and command
  implementations using multiple files with helper functions.
- **Test Plan:** Comprehensive unit and integration tests cover all main use cases and edge cases, ensuring robust
  behavior.
