# Plan

## CT Doc Command Help
- [x] internal/cli/cli.go: Expand doc command descriptions and error handling to spell out required owner, repository, and path flags while guiding users toward optional URL, reference, rules, and clipboard settings.
- [x] tests/doc_test.go: Add coverage asserting that missing coordinates produce actionable guidance and that command help enumerates required versus optional parameters.
- [x] README.md: Extend documentation with a doc command usage section listing required and optional parameters.
- [x] NOTES.md: Mark the CT doc help issue as completed once tests and guidance are in place.

## Logging Output Formatting
- [x] internal/utils: Provide a constructor that configures zap with console encoding and friendly defaults for CLI output.
- [x] main.go, cmd/ctx/main.go: Reuse the shared logger constructor so fatal errors emit human-readable text instead of JSON.
- [x] tests/doc_test.go: Ensure fatal error output remains detectable without relying on JSON formatting.
