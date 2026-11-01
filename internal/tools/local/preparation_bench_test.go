// Purpose: Measure candidate preparation time and allocations at actual bounded concurrency.
// Pattern: Fixed-input filesystem benchmark.
// Usage: Run BenchmarkCandidatePreparation with -benchmem; candidates never execute.
// Related: internal/tools/local/parallel.go, internal/tools/local/version.go.
package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/policy"
)

func benchmarkPrepareCandidates(e *Executor, calls []toolapi.Call, count int) ([]toolapi.Execution, error) {
	ready := make([]toolapi.Execution, len(calls))
	failures := make([]error, len(calls))
	prepare := func(i int) {
		ready[i], failures[i] = e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, calls[i])
	}
	if count == 1 {
		for i := range calls {
			prepare(i)
		}
	} else {
		var next atomic.Int64
		var workers sync.WaitGroup
		for range count {
			workers.Go(func() {
				for {
					i := int(next.Add(1)) - 1
					if i >= len(calls) {
						return
					}
					prepare(i)
				}
			})
		}
		workers.Wait()
	}
	return ready, errors.Join(failures...)
}

func BenchmarkCandidatePreparation(b *testing.B) {
	for _, lines := range []int{32, 8192, 23000} {
		b.Run(fmt.Sprintf("Lines%d", lines), func(b *testing.B) {
			root, home := b.TempDir(), b.TempDir()
			vault := &credentials.Vault{}
			e, err := NewExecutor(root, home, filepath.Join(home, ".meca"), vault.Redact)
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() {
				if err := e.Close(); err != nil {
					b.Error(err)
				}
			})
			var source strings.Builder
			for i := range lines {
				fmt.Fprintf(&source, "line %05d: unchanged source content\n", i)
			}
			before := source.String()
			calls := make([]toolapi.Call, 16)
			for i := range calls {
				path := fmt.Sprintf("file%d.txt", i)
				if err := os.WriteFile(filepath.Join(root, path), []byte(before), 0600); err != nil {
					b.Fatal(err)
				}
				body, err := json.Marshal(map[string]any{"path": path, "edits": []map[string]string{{"old_text": fmt.Sprintf("line %05d: unchanged source content", lines/2), "new_text": "changed source content"}}})
				if err != nil {
					b.Fatal(err)
				}
				calls[i] = call("edit_file", string(body))
			}
			for _, name := range []string{"Serial", "Bounded"} {
				b.Run(name, func(b *testing.B) {
					count := 1
					if name == "Bounded" {
						plan, err := e.ParallelBatch(context.Background(), policy.Settings{Scope: policy.System}, calls, 16)
						if err != nil {
							b.Fatal(err)
						}
						count = plan.PreparationConcurrency
					}
					b.ReportAllocs()
					for b.Loop() {
						ready, err := benchmarkPrepareCandidates(e, calls, count)
						if err != nil {
							b.Fatal(err)
						}
						runtime.KeepAlive(ready)
					}
				})
			}
		})
	}
}

func BenchmarkExecutionSnapshot(b *testing.B) {
	root := b.TempDir()
	body := []byte(strings.Repeat("source\n", 150000))
	if err := os.WriteFile(filepath.Join(root, "source"), body, 0600); err != nil {
		b.Fatal(err)
	}
	handle, err := os.OpenRoot(root)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := handle.Close(); err != nil {
			b.Error(err)
		}
	})
	path := target{absolute: filepath.Join(root, "source"), relative: "source", root: handle}
	b.ReportAllocs()
	for b.Loop() {
		if value, err := snapshot(path); err != nil || !value.exists {
			b.Fatalf("snapshot: %v", err)
		}
	}
}
