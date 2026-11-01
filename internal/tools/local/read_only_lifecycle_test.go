//go:build linux || darwin

// Purpose: Verify independent command registries, cancellation and capture cleanup for LAT forks.
// Pattern: Owned process lifecycle integration tests.
// Usage: go test ./internal/tools/local -run TestReadOnlyLifecycle.
// Related: internal/tools/local/read_only.go, internal/tools/local/process/registry.go.
package local

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestReadOnlyLifecycleIsolationAndCleanup(t *testing.T) {
	parent, project := testExecutor(t)
	child := readOnlyForTest(t, parent)
	sibling := readOnlyForTest(t, parent)
	child.commandWait = 0
	fifo := filepath.Join(project, "input")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	parentID, err := parent.commands.Start(context.Background(), project, "cat input", os.Environ(), "parent")
	if err != nil {
		t.Fatal(err)
	}
	result, err := readOnlyQuery(t, child, "cat input").Execute(context.Background(), false)
	if err != nil || result.Process == nil || result.Process.Status != "running" {
		t.Fatalf("start query: %+v %v", result, err)
	}
	id := result.Process.CommandID
	if id == "" || id == parentID {
		t.Fatalf("not independently owned: %q", id)
	}
	if _, err := sibling.commands.Cancel(context.Background(), id); err == nil {
		t.Fatal("sibling cancelled another fork")
	}
	if _, err := child.commands.Cancel(context.Background(), parentID); err == nil {
		t.Fatal("fork cancelled parent")
	}
	if len(child.CommandActivity()) != 1 || len(sibling.CommandActivity()) != 0 || len(parent.CommandActivity()) != 1 {
		t.Fatal("registries shared")
	}
	archive := child.artifacts
	if err := child.close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(archive); !os.IsNotExist(err) {
		t.Fatalf("capture directory retained: %v", err)
	}
	if state, err := parent.commands.Wait(context.Background(), parentID, 0); err != nil || state.Status != "running" {
		t.Fatalf("parent stopped: %+v %v", state, err)
	}
	if _, err := child.Prepare(context.Background(), defaultReadOnlySettings(), call("execute_command", `{"command":"pwd"}`)); err == nil {
		t.Fatal("closed fork accepted work")
	}
	if err := child.close(); err != nil {
		t.Fatal("cleanup was not idempotent", err)
	}
}

func TestReadOnlyLifecycleCancelTool(t *testing.T) {
	parent, project := testExecutor(t)
	child := readOnlyForTest(t, parent)
	child.commandWait = 0
	if err := syscall.Mkfifo(filepath.Join(project, "input"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := readOnlyQuery(t, child, "cat input").Execute(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	op, err := child.Prepare(context.Background(), defaultReadOnlySettings(), call("cancel_command", `{"command_id":"`+result.Process.CommandID+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cancelled, err := op.Execute(ctx, false)
	if err != nil || cancelled.Failed || cancelled.Process == nil || cancelled.Process.Status != "interrupted" {
		t.Fatalf("cancel: %+v %v", cancelled, err)
	}
	if states := child.commands.List(); len(states) != 1 || states[0].Status == "running" {
		t.Fatalf("process retained: %+v", states)
	}
}

func TestReadOnlyLifecycleCleanupDuringInitialWait(t *testing.T) {
	parent, project := testExecutor(t)
	child := readOnlyForTest(t, parent)
	child.commandWait = time.Minute
	if err := syscall.Mkfifo(filepath.Join(project, "input"), 0600); err != nil {
		t.Fatal(err)
	}
	op := readOnlyQuery(t, child, "cat input")
	done := make(chan error, 1)
	go func() { _, err := op.Execute(context.Background(), false); done <- err }()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for len(child.CommandActivity()) == 0 {
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatal("query did not start")
		}
	}
	if err := child.close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("active wait was not cancelled")
		}
	case <-deadline.C:
		t.Fatal("cleanup did not join active execution")
	}
}

func TestReadOnlyAcknowledgementReleasesCompletedContexts(t *testing.T) {
	parent, project := testExecutor(t)
	child := readOnlyForTest(t, parent)
	child.commandWait = 0
	if err := syscall.Mkfifo(filepath.Join(project, "pending"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := readOnlyQuery(t, child, "cat pending").Execute(context.Background(), false)
	if err != nil || result.Process == nil {
		t.Fatalf("start: %+v %v", result, err)
	}
	id := result.Process.CommandID
	if len(child.release) != 1 {
		t.Fatal("running query lost cancellation owner")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := child.commands.Cancel(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := child.AcknowledgeCommand(id); err != nil {
		t.Fatal(err)
	}
	if len(child.release) != 0 || len(child.CommandActivity()) != 0 {
		t.Fatal("completed query retained its context or registry entry")
	}
	if err := child.AcknowledgeCommand(id); err != nil {
		t.Fatal("repeated acknowledgement failed", err)
	}
}
