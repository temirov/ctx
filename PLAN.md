# Plan

## CT-16 GitHub Documentation Retrieval
- [x] internal/docs, tools/gh_to_doc: Promote the GitHub scraping logic into an injectable package and extend the documentation collector to support `relevant` (local) vs `full` (local + remote) modes.
- [x] internal/commands/doc.go, internal/cli/cli.go: Add the `doc`/`d` command, convert `--doc` into an enumerated flag, and wire remote documentation fetching with clipboard and configuration defaults.
- [x] internal/types/types.go, internal/config/app_config.go: Introduce documentation mode constants, update configuration schemas, and validate allowed values.
- [x] internal/services/mcp/server.go, internal/cli/mcp_executor.go, internal/cli/mcp_test.go: Advertise the new command, expose mode-aware parameters, and adapt MCP execution paths.
- [x] tests/ctx_test.go: Add integration coverage using stubbed GitHub responses for jspreadsheet, marked.js, and beer.css targets to prove full extraction.
- [x] README.md, NOTES.md: Document the new workflow, describe documentation modes, and mark CT-16 as complete.
