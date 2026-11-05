// Purpose: Verify path-specific wakeups and fair admission without serializing independent targets.
// Pattern: Deterministic concurrent admission regression tests.
// Usage: go test -race ./internal/tools/team/workspace.
// Related: internal/tools/team/workspace/workspace.go, internal/tools/team/workspace/waiters.go.
package workspace

import (
	"context"
	"testing"
	"testing/synctest"
)

func TestWaitingCommandIsNotOvertakenOnItsTarget(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := New()
		first, _ := w.ReserveCommand(t.Context(), []string{"/project/a"}, nil, nil)
		second := make(chan *Lease, 1)
		go func() {
			held, _ := w.ReserveCommand(t.Context(), []string{"/project/a"}, nil, nil)
			second <- held
		}()
		synctest.Wait()
		first.Release()
		ctx, cancel := context.WithCancel(t.Context())
		third, err := w.ReserveCommand(ctx, []string{"/project/a"}, nil, func() error { cancel(); return nil })
		if third != nil || err == nil {
			third.Release()
			t.Fatal("a new command overtook an already waiting mutation")
		}
		synctest.Wait()
		(<-second).Release()
	})
}

func TestOverlapChecksPathBoundariesWithoutAllocating(t *testing.T) {
	for _, pair := range [][2]string{{"/project/a", "/project/a/file"}, {"/", "/project"}, {"/project/a", "/project/ab"}} {
		a, b := target{path: pair[0]}, target{path: pair[1]}
		if allocs := testing.AllocsPerRun(100, func() { overlap(a, b) }); allocs != 0 {
			t.Fatal("path overlap allocated prefix strings", allocs)
		}
	}
	if overlap(target{path: "/project/a"}, target{path: "/project/ab"}) {
		t.Fatal("path prefix ignored the separator boundary")
	}
}

func BenchmarkReservationRelease(b *testing.B) {
	w := New()
	paths := []string{b.TempDir() + "/target"}
	b.ReportAllocs()
	for b.Loop() {
		held, err := w.ReserveCommand(b.Context(), paths, nil, nil)
		if err != nil {
			b.Fatal(err)
		}
		held.Release()
	}
}
