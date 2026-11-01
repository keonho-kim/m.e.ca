// Purpose: Verify real Git repository helper configuration cannot escape LAT query execution.
// Pattern: Native executable configuration regression test.
// Usage: go test ./internal/tools/local -run TestReadOnlyGit.
// Related: internal/tools/local/read_only_command.go, internal/policy/readonly/git.go.
package local

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadOnlyGitDisablesRepositoryHelpers(t *testing.T) {
	git, err := readOnlyExecutable("git")
	if err != nil {
		t.Skip(err)
	}
	parent, project := testExecutor(t)
	child := readOnlyForTest(t, parent)
	setup := func(args ...string) {
		t.Helper()
		command := exec.Command(git, args...)
		command.Dir = project
		command.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + t.TempDir(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null"}
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git setup: %s %v", output, err)
		}
	}
	setup("init", "--quiet")
	tracked := filepath.Join(project, "tracked.txt")
	if err := os.WriteFile(tracked, []byte("before\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".gitattributes"), []byte("*.txt diff=query\n"), 0600); err != nil {
		t.Fatal(err)
	}
	setup("add", "tracked.txt", ".gitattributes")
	if err := os.WriteFile(tracked, []byte("after\n"), 0600); err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(project, "query-helper")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\nprintf escaped > '"+filepath.Join(project, "escaped")+"'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	config, err := os.OpenFile(filepath.Join(project, ".git", "config"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := config.WriteString("\n[core]\n fsmonitor = " + helper + "\n pager = " + helper + "\n[diff]\n external = " + helper + "\n[diff \"query\"]\n textconv = " + helper + "\n")
	closeErr := config.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatalf("config: %v %v", writeErr, closeErr)
	}
	for _, query := range []string{"git status --short", "git diff -- tracked.txt", "git ls-files", "git rev-parse --show-toplevel"} {
		result, err := readOnlyQuery(t, child, query).Execute(context.Background(), true)
		if err != nil || result.Failed {
			t.Fatalf("%s: %+v %v", query, result, err)
		}
		if query == "git diff -- tracked.txt" && !strings.Contains(string(result.Content), "after") {
			t.Fatalf("missing ordinary diff: %s", result.Content)
		}
	}
	if _, err := os.Stat(filepath.Join(project, "escaped")); !os.IsNotExist(err) {
		t.Fatal("repository configured execution helper ran")
	}
}
