//go:build linux || darwin

// Purpose: Verify a waiting read cannot serialize independent LAT queries or shutdown.
// Pattern: FIFO-controlled process lifecycle regression.
// Usage: Run TestReadOnlyQueriesOverlap with the race detector.
// Related: internal/tools/local/read_only.go, internal/tools/local/read_only_lifetime.go.
package local

import (
	"context"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestReadOnlyQueriesOverlapAndCloseJoinsThem(t *testing.T) {
	parent, project := testExecutor(t)
	child := readOnlyForTest(t, parent)
	child.commandWait = time.Minute
	if err := syscall.Mkfifo(filepath.Join(project, "pending"), 0600); err != nil {
		t.Fatal(err)
	}
	waiting := make(chan error, 1)
	blocked := readOnlyQuery(t, child, "cat pending")
	go func() {
		_, err := blocked.Execute(context.Background(), false)
		waiting <- err
	}()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for len(child.CommandActivity()) == 0 {
		select {
		case <-ticker.C:
		case <-deadline.C:
			child.close()
			<-waiting
			t.Fatal("waiting query did not start")
		}
	}
	ready := make(chan error, 1)
	op := readOnlyQuery(t, child, "pwd")
	go func() { _, err := op.Execute(context.Background(), false); ready <- err }()
	select {
	case err := <-ready:
		if err != nil {
			t.Fatal(err)
		}
	case <-deadline.C:
		child.close()
		<-waiting
		<-ready
		t.Fatal("an independent query waited behind the blocked read")
	}
	if err := child.close(); err != nil {
		t.Fatal(err)
	}
	if err := <-waiting; err == nil {
		t.Fatal("shutdown did not cancel the active wait")
	}
	if _, err := op.Execute(context.Background(), false); err == nil {
		t.Fatal("closed executor accepted another invocation")
	}
}
