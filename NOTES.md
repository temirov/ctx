# Notes

## Rules of engagement

Review the notes.md. Make a plan for autonomously fixing every bug. Ensure no regressions. Ensure adding tests. Lean into integration tests. Fix every bug.

Fix bugs one by one. Write a nice comprehensive commit message AFTER EACH bug is fixed and tested and covered with tests. Do not work on all bugs at all. Work at one bug at a time. 

Remove a bug from the notes.md after the bug is fixed. commit and push to the remote.

Leave Bugfix section empty but dont delete the section itself.

## BugFixes

### Top level summary

Add top level sumary. add a boolean flag --summary that shows the summary for all ot the resulting files:

total files
total file size

the flag is true by default but can be turned off with --summmary false

Add top level the summary to the output of both the tree command and content command. it shall show the number of resulting files (factor in excludes, .ignore/.gitignore and only give summary for the actual files), and files size. it shall work for both json and xml outputs.

Note that the summary is recursive and is provided at every level of the folder hierearchy assuming the folder is included. Summary details doesdnt have its own key, it is a part of the folders payload

Example: 
"path": "/Users/tyemirov/Development/ctx/tests",
"name": "tests",
"type": "directory",
"lastModified": "2025-10-05 11:56",
"totalFiles": 1,
"totalSize": "43kb"
"children": []

Keys:

"path"
"name"
"type"
"lastModified"
"totalFiles"
"totalSize"

- For JSON/XML results (tree and content), we should no longer wrap data in top-level summary/code objects.
- Instead, the root element/node should represent the starting folder itself—with fields path, name, type, lastModified, totalFiles, totalSize, and children.
- Every directory node should follow that exact shape recursively. the children key remains, as the entry to the recursive structure.
- Documentation is updated to refelct the summary capability and the changed format of output.

The summary is not included for the files and is relevant for the folders only

### Exclude files after shell expansion

12:25:54 tyemirov@Vadyms-MacBook-Pro:~/Development/Research/tokens - [] $ ctx t
[
  {
    "path": "/Users/tyemirov/Development/Research/tokens",
    "name": "tokens",
    "type": "directory",
    "children": [
      {
        "path": "/Users/tyemirov/Development/Research/tokens/README.md",
        "name": "README.md",
        "type": "file",
        "mimeType": "text/plain; charset=utf-8"
      },
      {
        "path": "/Users/tyemirov/Development/Research/tokens/anthropic_count.py",
        "name": "anthropic_count.py",
        "type": "file",
        "mimeType": "text/plain; charset=utf-8"
      },
      {
        "path": "/Users/tyemirov/Development/Research/tokens/go.mod",
        "name": "go.mod",
        "type": "file",
        "mimeType": "text/plain; charset=utf-8"
      },
      {
        "path": "/Users/tyemirov/Development/Research/tokens/go.sum",
        "name": "go.sum",
        "type": "file",
        "mimeType": "text/plain; charset=utf-8"
      },
      {
        "path": "/Users/tyemirov/Development/Research/tokens/llama_count.py",
        "name": "llama_count.py",
        "type": "file",
        "mimeType": "text/plain; charset=utf-8"
      },
      {
        "path": "/Users/tyemirov/Development/Research/tokens/main.go",
        "name": "main.go",
        "type": "file",
        "mimeType": "text/plain; charset=utf-8"
      }
    ]
  }
]

ctx c -e go.*  returns a single file, go.sum

I expect both go.sum and go.mod to be exluded