# Content

Content is a command‑line tool written in Go that displays a directory tree view or outputs file contents for one or
more specified directories. It supports exclusion patterns via **.ignore** and **.gitignore** files within each
directory, as well as an optional global exclusion flag.

## Features

- **Multi-Directory Processing:** Accepts one or more directory paths as input.
- **Tree Command:** Recursively displays the directory structure for each specified directory in a tree‑like format,
  clearly separated.
- **Content Command:** Outputs the concatenated contents of files from all specified directories.
- **Exclusion Patterns:**
    - Reads patterns from an **.ignore** file in the root of *each* processed directory (can be disabled with
      `--no-ignore`).
    - Reads patterns from a **.gitignore** file by default in the root of *each* processed directory (can be disabled
      with `--no-gitignore`).
    - Patterns from `.ignore` and `.gitignore` within a specific directory are combined, deduplicated, and applied only
      to that directory's processing.
- **Exclusion Flags:**
    - **-e / --e:** Excludes a designated folder name if it appears as a direct child within *any* of the specified root
      directories. Both `-e` and `--e` work.
    - **--no-gitignore:** Disables reading from **.gitignore** files in all processed directories.
    - **--no-ignore:** Disables reading from the **.ignore** files in all processed directories.
- **Command Abbreviations:**
    - `t` is an alias for `tree`.
    - `c` is an alias for `content`.
- **Deduplication:** Duplicate input directory paths (after resolving to absolute paths) are processed only once.

## Installation

1. **Ensure you have Go installed:**
   Download and install Go from [golang.org](https://golang.org/dl/).

2. **Set up your environment:**
   Make sure your `GOBIN` (or `$GOPATH/bin`) is in your system's `PATH`.

3. **Install the utility:**
   Run the following command to install the latest version:

   ```bash
   go install github.com/temirov/content@latest
   ```

4. **Verify installation:**
   Confirm that the `content` binary is available in your `GOBIN` directory by running:

   ```bash
   content --help
   # Or just 'content' to see usage
   ```

## Usage

The basic syntax for using the utility is:

```bash
content <tree|t|content|c> [directory1] [directory2] ... [-e|--e exclusion_folder] [--no-gitignore] [--no-ignore]
```

### Commands

- **tree (or t):**
  Displays a directory tree view for each specified directory.
- **content (or c):**
  Outputs the concatenated contents of files from all specified directories.

### Arguments

- **directory1, directory2, ...:**
  Optional. One or more paths to the root directories you want to process.
    - If no directories are provided, it defaults to the current directory (`"."`).
    - Paths can be relative or absolute.
    - Duplicate paths are ignored.
- **-e or --e exclusion_folder:**
  Optional flag. Specifies a single folder *name* to exclude globally.
    - If a folder with this exact name exists directly under any of the specified `directory` arguments, it will be
      entirely excluded from processing for that specific directory.
    - Example: `content c proj1 proj2 -e node_modules` excludes `proj1/node_modules` and `proj2/node_modules`.
    - This only applies to direct children; nested folders with the same name are *not* affected by this flag (though
      they might be by `.ignore`/`.gitignore` rules).
- **--no-gitignore:**
  Optional flag. Disables loading of `.gitignore` files from all processed directories.
- **--no-ignore:**
  Optional flag. Disables loading of `.ignore` files from all processed directories.

### Examples

- **Display tree views for `projectA` and `projectB`, excluding `dist` folders in both:**

  ```bash
  content tree projectA projectB -e dist
  ```

- **Get concatenated content of files in the current directory and `/path/to/libs`, excluding `logs` and
  ignoring `.gitignore` files:**

  ```bash
  content c . /path/to/libs -e logs --no-gitignore
  ```

- **Get content of `src` directory, ignoring patterns from its `.ignore` file:**

  ```bash
  content content src --no-ignore
  ```

- **Display tree of the current directory (default):**

  ```bash
  content t
  ```

## Configuration

The utility loads exclusion patterns from configuration files found within *each* directory being processed:

1. **.ignore**
    - Located at the root of a processed directory (e.g., `projectA/.ignore`).
    - Uses standard Gitignore pattern syntax.
    - Patterns only apply to the contents of that specific directory.
    - Loading can be disabled globally using `--no-ignore`.
    - Example `projectA/.ignore`:
      ```plaintext
      # Project A specific ignores
      build/
      *.tmp
      ```

2. **.gitignore**
    - Located at the root of a processed directory (e.g., `projectA/.gitignore`).
    - Read by default.
    - Patterns only apply to the contents of that specific directory.
    - Loading can be disabled globally using `--no-gitignore`.

Patterns from both files (if loaded) within the same directory are combined and deduplicated for processing that
directory.

## Development

### Running Tests

From the project root, run:

```bash
go test ./...
```

### Building Locally

To build the binary locally, run:

```bash
go build -o content .
```

## Contributing

Contributions are welcome. To contribute:

1. Fork the repository.
2. Create a feature branch.
3. Commit your changes.
4. Submit a pull request.

## License

This project is licensed under the [MIT License](LICENSE).