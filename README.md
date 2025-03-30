# Content

Content is a command‑line tool written in Go that displays a directory tree view or outputs file contents for one or
more specified files and/or directories. It supports exclusion patterns via **.ignore** and **.gitignore** files within
each directory, as well as an optional global exclusion flag.

## Features

- **Mixed File/Directory Processing:** Accepts one or more file and/or directory paths as input.
- **Tree Command:**
    - Recursively displays the directory structure for each specified directory in a tree‑like format, clearly
      separated.
    - Lists explicitly provided file paths prefixed with `[File]`.
- **Content Command:**
    - Outputs the content of explicitly provided files.
    - Outputs the concatenated contents of files found within specified directories (respecting ignores).
- **Exclusion Patterns:**
    - Reads patterns from an **.ignore** file in the root of *each* processed **directory** (can be disabled with
      `--no-ignore`).
    - Reads patterns from a **.gitignore** file by default in the root of *each* processed **directory** (can be
      disabled with `--no-gitignore`).
    - Patterns from `.ignore` and `.gitignore` within a specific directory are combined, deduplicated, and applied
      **only** during the traversal of that directory.
    - **Important:** Ignore rules *do not* filter out explicitly listed file paths. If you provide
      `path/to/ignored_file.txt` as an argument, its content will be shown by the `content` command.
- **Exclusion Flags:**
    - **-e / --e:** Excludes a designated folder name if it appears as a direct child within *any* of the specified
      root **directories**. Both `-e` and `--e` work. Has no effect on explicitly listed files.
    - **--no-gitignore:** Disables reading from **.gitignore** files in all processed directories.
    - **--no-ignore:** Disables reading from the **.ignore** files in all processed directories.
- **Command Abbreviations:**
    - `t` is an alias for `tree`.
    - `c` is an alias for `content`.
- **Deduplication:** Duplicate input paths (after resolving to absolute paths) are processed only once.

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
content <tree|t|content|c> [path1] [path2] ... [-e|--e exclusion_folder] [--no-gitignore] [--no-ignore]
```

### Commands

- **tree (or t):**
  Displays a directory tree view for each specified directory and lists specified files.
- **content (or c):**
  Outputs the content of specified files and the concatenated contents of files within specified directories.

### Arguments

- **path1, path2, ...:**
  Optional. One or more paths (files or directories) you want to process.
    - If no paths are provided, it defaults to the current directory (`"."`).
    - Paths can be relative or absolute.
    - Duplicate paths (after resolution) are ignored.
- **-e or --e exclusion_folder:**
  Optional flag. Specifies a single folder *name* to exclude globally during **directory** traversal.
    - If a folder with this exact name exists directly under any of the specified **directory** arguments, it will be
      entirely excluded from processing for that specific directory.
    - Example: `content c proj1 proj2 -e node_modules` excludes `proj1/node_modules` and `proj2/node_modules` during
      traversal.
    - This only applies to direct children of specified directories; nested folders with the same name are *not*
      affected by this flag.
    - This flag has **no effect** on explicitly listed **file** arguments.
- **--no-gitignore:**
  Optional flag. Disables loading of `.gitignore` files from all processed directories.
- **--no-ignore:**
  Optional flag. Disables loading of `.ignore` files from all processed directories.

### Examples

- **Display tree for `projectA` dir and list `config.yaml` file, excluding `dist` folders within `projectA`:**

  ```bash
  content tree projectA config.yaml -e dist
  ```

- **Get content of `main.go`, `utils/helpers.go`, and all files within the `pkg` directory, excluding `pkg/logs`:**

  ```bash
  content c main.go utils/helpers.go pkg -e logs
  ```

- **Get content of `README.md` and files in current dir, ignoring `.gitignore` rules during directory scan:**

  ```bash
  content c README.md . --no-gitignore
  ```

- **Get content of an explicitly listed file, even if ignored by `.ignore`:**
  *(Assume `my_dir/.ignore` contains `secret.txt`)*

  ```bash
  # This WILL print secret.txt because it's listed explicitly
  content c my_dir/secret.txt

  # This will NOT print secret.txt (if found during traversal)
  content c my_dir
  ```

- **Display tree of the current directory (default):**

  ```bash
  content t
  ```

## Configuration (Applies ONLY during Directory Traversal)

Exclusion patterns are loaded from configuration files found within *each* **directory** being processed:

1. **.ignore**
    - Located at the root of a processed directory (e.g., `projectA/.ignore`).
    - Uses standard Gitignore pattern syntax.
    - Patterns only apply when traversing the contents of that specific directory.
    - Loading can be disabled globally using `--no-ignore`.

2. **.gitignore**
    - Located at the root of a processed directory (e.g., `projectA/.gitignore`).
    - Read by default.
    - Patterns only apply when traversing the contents of that specific directory.
    - Loading can be disabled globally using `--no-gitignore`.

Patterns from both files (if loaded) within the same directory are combined and deduplicated for processing that
directory traversal. **These patterns do not affect explicitly listed file arguments.**

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