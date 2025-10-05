# Notes

## Rules of engagement

Review the notes.md. Make a plan for autonomously fixing every bug. Ensure no regressions. Ensure adding tests. Lean into integration tests. Fix every bug.

Fix bugs one by one. Write a nice comprehensive commit message AFTER EACH bug is fixed and tested and covered with tests. Do not work on all bugs at all. Work at one bug at a time sequntially. 

Remove a bug from the notes.md after the bug is fixed. commit and push to the remote.

Leave BugFixes section empty but don't delete the section itself.

## BugFixes

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
