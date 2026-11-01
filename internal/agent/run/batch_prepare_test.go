// Purpose: Verify candidate preparation stays bounded, joined and non-executing on failure.
// Pattern: Channel-controlled runtime integration tests.
// Usage: Run with the race detector for the file preparation boundary.
// Related: internal/agent/run/batch_prepare.go, internal/tools/local/parallel.go.
package run

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/local"
)

type preparingTools struct {
	*local.Executor
	entered         chan string
	release         chan struct{}
	active, maximum atomic.Int32
	fail            bool
}

func (e *preparingTools) Prepare(ctx context.Context, settings policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	active := e.active.Add(1)
	defer e.active.Add(-1)
	for old := e.maximum.Load(); active > old; old = e.maximum.Load() {
		if e.maximum.CompareAndSwap(old, active) {
			break
		}
	}
	e.entered <- call.ID
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-e.release:
	}
	if e.fail && (call.ID == "1" || call.ID == "3") {
		return nil, fmt.Errorf("invalid candidate %s", call.ID)
	}
	return e.Executor.Prepare(ctx, settings, call)
}

func TestParallelPreparationBoundsFailureAndCancellation(t *testing.T) {
	for _, mode := range []string{"success", "failure", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			var calls []toolapi.Call
			for i := range 8 {
				calls = append(calls, batchCall(fmt.Sprint(i), "write_file", map[string]any{"path": fmt.Sprintf("file%d", i), "content": "body"}))
			}
			r, tools, root, journal, _ := realBatchRunner(t, calls)
			preparing := &preparingTools{Executor: tools, entered: make(chan string, 8), release: make(chan struct{}), fail: mode == "failure"}
			r.Tools = preparing
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			done := make(chan struct{})
			var runErr error
			go func() { defer close(done); _, runErr = r.Run(ctx, "create files", approve, emit) }()
			t.Cleanup(func() { cancel(); <-done })
			for range 4 {
				select {
				case <-preparing.entered:
				case <-ctx.Done():
					t.Fatal("preparation did not overlap")
				}
			}
			if mode == "cancel" {
				cancel()
			} else {
				close(preparing.release)
			}
			<-done
			err := runErr
			cancel()
			if mode == "cancel" {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if preparing.active.Load() != 0 || preparing.maximum.Load() != 4 {
				t.Fatal("preparation workers escaped their bound or lifetime")
			}
			for i := range calls {
				_, err := os.Stat(filepath.Join(root, fmt.Sprintf("file%d", i)))
				if (err == nil) != (mode == "success") {
					t.Fatalf("unexpected effect in %s: %v", mode, err)
				}
			}
			if mode == "failure" {
				failed := ""
				for _, event := range journal.events {
					if event.Kind == "tool_failed" && event.CallID == "1" && strings.Contains(event.Text, "invalid candidate 1") {
						failed = event.CallID
					}
				}
				if failed != "1" {
					t.Fatal("failure selection lost call order")
				}
			}
		})
	}
}
