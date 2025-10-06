# Notes

## Rules of engagement

Review the notes.md. Make a plan for autonomously fixing every bug. Ensure no regressions. Ensure adding tests. Lean into integration tests. Fix every bug.

Fix bugs one by one. Write a nice comprehensive commit message AFTER EACH bug is fixed and tested and covered with tests. Do not work on all bugs at all. Work at one bug at a time sequntially. 

Remove a bug from the notes.md after the bug is fixed. commit and push to the remote.

Leave BugFixes section empty but don't delete the section itself.

## BugFixes

### Tokenizer

1. to get the tokens for claude we need to reach to official claude API messages.count_tokens, which is free but requries the API_KEY environment variable. we shall us that method and document the requirements in the README. if there is no env variable ANTHROPIC_API_KEY we shall fail from Go not Python, and provide a helpful message
