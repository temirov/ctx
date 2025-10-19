package tests

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestPlanFileNotTracked(t *testing.T) {
	checkContext, cancelCheck := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelCheck()
	gitCommand := exec.CommandContext(checkContext, "git", "ls-files", "--error-unmatch", "PLAN.md")
	runError := gitCommand.Run()
	if runError == nil {
		t.Fatalf("PLAN.md must remain untracked; remove it from the repository")
	}
	if checkContext.Err() == context.DeadlineExceeded {
		t.Fatalf("git ls-files timed out while verifying PLAN.md tracking status")
	}
}
