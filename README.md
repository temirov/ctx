# Content

Content is a command‑line tool written in Go that displays a directory tree view or outputs file contents. It supports
exclusion patterns via a `.contentignore` file (using full Git‑ignore semantics) and an optional additional exclusion
flag (`-e`) that omits directories directly under the working directory.

## Features

- **Tree Command:** Recursively displays the directory structure in a tree-like format.
- **Content Command:** Outputs the contents of files within a specified directory.
- **Exclusion Patterns:** Uses a `.contentignore` file to exclude files and directories based on glob patterns.
- **Optional `-e` Flag:** Excludes a designated folder (only when it is a direct child of the working directory).

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
content <command> [root_directory] [-e exclusion_folder]
```

### Commands

- **tree**  
  Displays a directory tree view.

- **content**  
  Outputs the contents of files under a specified directory.

### Arguments

- **root_directory**  
  Optional. Defaults to the current directory (`"."`) if not provided.

- **-e exclusion_folder**  
  Optional flag. Specifies a single additional folder (relative to the working directory) to exclude.
    - When the folder is directly under the working directory (or provided root), it is entirely excluded from
      processing.
    - Nested folders with the same name are not excluded.

### Examples

- **Display a tree view of the current directory while excluding the `log` folder:**

  ```bash
  content tree -e log
  ```

- **Display a tree view of a specific directory (`pkg`) while excluding the `log` folder:**

  ```bash
  content tree pkg -e log
  ```

- **Output file contents from the current directory while excluding the `log` folder:**

  ```bash
  content content -e log
  ```

- **Output file contents from a specific directory (`pkg`) while excluding the `log` folder:**

  ```bash
  content content pkg -e log
  ```

## Configuration

The utility reads exclusion patterns from a `.contentignore` file located in your current working directory. The file
uses Git‑ignore semantics where:

- Each non-empty line (that does not start with `#`) is treated as a glob pattern.
- For example:

  ```plaintext
  # Exclude log directories and temporary files
  log/
  *.tmp
  ```

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
[.gitignore](../notify/.gitignore)