# Content

Content is a command‑line tool written in Go that displays a directory tree view or outputs file contents. It supports
exclusion patterns via an **.ignore** file (using full Git‑ignore semantics) and a **.gitignore** file by default, as
well as an optional exclusion flag.

## Features

- **Tree Command:** Recursively displays the directory structure in a tree‑like format.
- **Content Command:** Outputs the contents of files within a specified directory.
- **Exclusion Patterns:**
    - Reads patterns from an **.ignore** file in the working directory (instead of the previous .ignore).
    - Reads patterns from a **.gitignore** file by default if it is present.
    - The patterns from both files are combined and deduplicated.
- **Exclusion Flags:**
    - **-e / --e:** Excludes a designated folder (only when it is a direct child of the working directory). Both `-e`
      and `--e` work.
    - **--no-gitignore:** Disables reading from the **.gitignore** file.
    - **--no-ignore:** Disables reading from the **.ignore** file.
- **Command Abbreviations:**
    - `t` is an alias for `tree`.
    - `c` is an alias for `content`.

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
   ```

## Usage

The basic syntax for using the utility is:

```bash
content <tree|t|content|c> [root_directory] [-e|--e exclusion_folder] [--no-gitignore] [--no-ignore]
```

### Commands

- **tree (or t):**  
  Displays a directory tree view.

- **content (or c):**  
  Outputs the contents of files under a specified directory.

### Arguments

- **root_directory:**  
  Optional. Defaults to the current directory (`"."`) if not provided.

- **-e or --e exclusion_folder:**  
  Optional flag. Specifies a single additional folder (relative to the working directory) to exclude.
    - When the folder is directly under the working directory (or provided root), it is entirely excluded from
      processing.
    - Nested folders with the same name are not excluded.

- **--no-gitignore:**  
  Optional flag. Disables loading of the **.gitignore** file.

- **--no-ignore:**  
  Optional flag. Disables loading of the **.ignore** file.

### Examples

- **Display a tree view of the current directory while excluding the `log` folder:**

  ```bash
  content tree -e log
  ```

- **Display a tree view of a specific directory (`pkg`) while excluding the `log` folder:**

  ```bash
  content t pkg --e log
  ```

- **Output file contents from the current directory while excluding the `log` folder:**

  ```bash
  content content -e log
  ```

- **Output file contents from a specific directory (`pkg`) while excluding the `log` folder and disabling .gitignore
  logic:**

  ```bash
  content c pkg -e log --no-gitignore
  ```

## Configuration

The utility loads exclusion patterns from two files by default:

1. **.ignore**
    - This file uses full Git‑ignore semantics.
    - Each non‑empty line (that does not start with `#`) is treated as a glob pattern.
    - For example:

      ```plaintext
      # Exclude log directories and temporary files
      log/
      *.tmp
      ```

2. **.gitignore**
    - This file is read by default and its patterns are combined with those from **.ignore**.
    - To disable its usage, run the utility with the `--no-gitignore` flag.

Additionally, if you do not wish to use the **.ignore** file, you can disable it with the `--no-ignore` flag.

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
