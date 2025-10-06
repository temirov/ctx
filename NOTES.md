# Notes

## Rules of engagement

Review the notes.md. Make a plan for autonomously fixing every bug. Ensure no regressions. Ensure adding tests. Lean into integration tests. Fix every bug.

Fix bugs one by one. Write a nice comprehensive commit message AFTER EACH bug is fixed and tested and covered with tests. Do not work on all bugs at all. Work at one bug at a time sequntially. 

Remove a bug from the notes.md after the bug is fixed. commit and push to the remote.

Leave BugFixes section empty but don't delete the section itself.

## BugFixes

### Tokenizer

go run main.go c --tokens --model llama-4.1
Error: python module sentencepiece not available; install it in your environment
{"level":"fatal","ts":1759730088.94082,"caller":"ctx/main.go:19","msg":"application execution failed","error":"python module sentencepiece not available; install it in your environment","stacktrace":"main.main\n\t/Users/tyemirov/Development/ctx/main.go:19\nruntime.main\n\t/usr/local/opt/go/libexec/src/runtime/proc.go:285"}
exit status 1

Change the invokation of python scripts through uv (use shebang syntax). document that for the tokenizer to work with such models as llama or claude, a uv utility must be available

an example of the invocation

```python
#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.11"
# dependencies = [
#   "opencv-python-headless>=4.9",
#   "numpy>=1.26",
#   "pillow>=10",
#   "pillow-heif>=0.18"
# ]
# ///
```