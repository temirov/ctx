# Plan

## CT-13 MCP Server Execution Support
- [x] Introduce a command executor registry and `/commands/<name>` endpoint so MCP clients can invoke tree/content/callchain remotely with structured responses.
- [x] Reuse existing CLI runners through executor adapters that normalize flags, capture stdout/stderr buffers, and honor documentation/token settings.
- [x] Add server-level unit tests for successful execution, missing commands, and bubbled status codes.
- [x] Add CLI-level end-to-end tests covering documentation retrieval for Go/JS/Python files in directories outside the current repository.
- [x] Update README with MCP invocation examples, including enabling documentation in HTTP requests.
