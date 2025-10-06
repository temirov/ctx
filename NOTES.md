# Notes

## Rules of engagement

Review the notes.md. Make a plan for autonomously fixing every bug. Ensure no regressions. Ensure adding tests. Lean into integration tests. Fix every bug.

Fix bugs one by one. Write a nice comprehensive commit message AFTER EACH bug is fixed and tested and covered with tests. Do not work on all bugs at all. Work at one bug at a time sequntially. 

Remove a bug from the notes.md after the bug is fixed. commit and push to the remote.

Leave BugFixes section empty but don't delete the section itself.

## BugFixes

### Streaming

Go has ways to **stream JSON as it’s being formed**, without building the whole object tree in memory.

The standard library `encoding/json` provides `json.Encoder`, which writes JSON tokens incrementally to an `io.Writer`. You can combine it with `http.ResponseWriter` or any stream.

### Example: streaming array elements

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

### Example: streaming JSON objects (newline-delimited JSON, aka NDJSON)

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

