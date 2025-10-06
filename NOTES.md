# Notes

## Rules of engagement

Review the NOTES.md. Make a plan for autonomously fixing every item under Features, BugFixes, Improvements, Maintenance. Ensure no regressions. Ensure adding tests. Lean into integration tests. Fix every issue. Document the changes.

Fix issues one by one. Write a nice comprehensive commit message AFTER EACH issue is fixed and tested and covered with tests. Do not work on all issues at all. Work at one issue at a time sequntially. 

Remove an issue from the NOTES.md after the issue is fixed (new tests are passing). Commit the changes and push to the remote.

Leave Features, BugFixes, Improvements, Maintenance sections empty when all fixes are implemented but don't delete the sections themselves.

## Features

## Improvements

## Maintenance

### Portability

- [ ] Consider how would the compiled Go executable run python scripts? Do we need to embed and recreate them at the runtime?

- [ ] Check OS specific assumptions. Can we run the compiled version of this app on Windows? Prepare a plan of what needs to change to be able to run it on Windows.

## BugFixes

### Streaming

There is no streaming, the results are pre-aggeregated and then displayed at once. Consider per-file streaming, with final statistics being below children nodes.

#### Schema

Ensure the schema below in the output
```shell
14:48:42 tyemirov@Vadyms-MacBook-Pro:~/Development/ctx - [(HEAD detached at e2a1435)] $ ANTHROPIC_API_KEY=<REDACTED> go run main.go c --tokens --model claude-4 cmd
```
```json
{
  "path": "/Users/tyemirov/Development/ctx/cmd",
  "name": "cmd",
  "type": "directory",
  "lastModified": "2025-09-04 13:49",
  "model": "claude-sonnet-4-20250514",
  "children": [
    {
      "path": "/Users/tyemirov/Development/ctx/cmd/ctx",
      "name": "ctx",
      "type": "directory",
      "lastModified": "2025-09-06 14:04",
      "model": "claude-sonnet-4-20250514",
      "children": [
        {
          "path": "/Users/tyemirov/Development/ctx/cmd/ctx/main.go",
          "name": "main.go",
          "type": "file",
          "size": "616b",
          "lastModified": "2025-09-06 14:04",
          "mimeType": "text/plain; charset=utf-8",
          "tokens": 207,
          "model": "claude-sonnet-4-20250514",
          "content": "package main\n\nimport (\n\t\"fmt\"\n\n\t\"github.com/temirov/ctx/internal/cli\"\n\t\"github.com/temirov/ctx/internal/utils\"\n\t\"go.uber.org/zap\"\n)\n\n// main is the entry point for the ctx command.\nfunc main() {\n\tloggerInstance, loggerInitializationError := zap.NewProduction()\n\tif loggerInitializationError != nil {\n\t\tpanic(fmt.Errorf(utils.LoggerInitializationFailedMessageFormat, loggerInitializationError))\n\t}\n\tdefer loggerInstance.Sync()\n\tif applicationExecutionError := cli.Execute(); applicationExecutionError != nil {\n\t\tloggerInstance.Fatal(utils.ApplicationExecutionFailedMessage, zap.Error(applicationExecutionError))\n\t}\n}\n"
        }
      ],
      "totalFiles": 1,
      "totalSize": "616b",
      "totalTokens": 207
    }
  ],
  "totalFiles": 1,
  "totalSize": "616b",
  "totalTokens": 207
}
```

#### Implementation

Similar approach will be available for XML

Go has ways to **stream JSON as it’s being formed**, without building the whole object tree in memory.

The standard library `encoding/json` provides `json.Encoder`, which writes JSON tokens incrementally to an `io.Writer`. You can combine it with `http.ResponseWriter` or any stream.

##### Example: streaming array elements

```go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func handler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(writer)

	// Begin array manually
	fmt.Fprint(writer, "[")

	// Stream items
	for index := 1; index <= 5; index++ {
		if index > 1 {
			fmt.Fprint(writer, ",")
		}
		encoder.Encode(map[string]int{"number": index})
		writer.(http.Flusher).Flush() // send chunk immediately
	}

	// End array
	fmt.Fprint(writer, "]")
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
```

Here:

* Each `encoder.Encode(...)` writes one JSON object.
* `Flusher.Flush()` pushes the chunk to the client immediately.
* You manage delimiters (`[`, `,`, `]`) yourself if building arrays.

##### Example: streaming JSON objects (newline-delimited JSON, aka NDJSON)

```go
package main

import (
	"encoding/json"
	"net/http"
)

func handler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/x-ndjson")
	encoder := json.NewEncoder(writer)

	for index := 1; index <= 5; index++ {
		encoder.Encode(map[string]int{"number": index})
		writer.(http.Flusher).Flush()
	}
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
```

This produces one JSON object per line, a common streaming format.

---

✅ Use `json.Encoder` with any `io.Writer`.
✅ Flush often if streaming over HTTP.
✅ For large payloads, NDJSON is simplest, but you can also build proper arrays by writing delimiters manually.
