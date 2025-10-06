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

### Format the output

- [ ] Add carriage return after the output
    - current behaviour: 
    ```shell
    </node>14:38:38 tyemirov@Vadyms-MacBook-Pro:~/Development/ctx - [summary] $ 
    ```
    - desired behaviour: 
    ```shell
    </node>
    14:38:38 tyemirov@Vadyms-MacBook-Pro:~/Development/ctx - [summary] $ 
    ```