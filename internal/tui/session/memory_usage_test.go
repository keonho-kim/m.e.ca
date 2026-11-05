// Purpose: Verify memory placement and cached display changes across terminal screens.
// Pattern: Terminal behavior tests with a controlled native-reader boundary.
// Usage: go test ./internal/tui/session -run TestMemory.
// Related: internal/tui/session/memory_usage.go, internal/tui/session/event_loop.go, internal/tui/view/memory.go.
package session

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gdamore/tcell/v3"
)

type memoryFrameScreen struct {
	tcell.Screen
	frames atomic.Int64
}

func (s *memoryFrameScreen) Show() { s.frames.Add(1); s.Screen.Show() }

func TestMemoryCounterPlacementAcrossScreens(t *testing.T) {
	a := screenFixture(t)
	a.memory.read = func() (uint64, error) { return 42 << 20, nil }
	a.memory.sample()
	for _, page := range []string{"chat", "model", "language", "permission", "files"} {
		a.page = page
		a.render()
		line := screenLine(a.screen, 0, 80)
		if !strings.Contains(line, "MEM. USAGE 42.0 MiB · 입력 0 · 출력 0") {
			t.Fatalf("%s header=%q", page, line)
		}
	}
	a.page = "chat"
	a.work.state, a.work.compressing = workerAgent, true
	a.render()
	if line := screenLine(a.screen, 0, 80); !strings.Contains(line, a.language.Text("Compress Memory")) || !strings.Contains(line, "MEM. USAGE 42.0 MiB") {
		t.Fatalf("compression or memory disappeared: %q", line)
	}
	a.screen.SetSize(32, 12)
	a.render()
	if line := screenLine(a.screen, 0, 32); !strings.Contains(line, a.language.Text("Compress Memory")) || strings.Contains(line, "MEM.") {
		t.Fatalf("narrow header lost the compression indicator: %q", line)
	}
	a.work.state, a.work.compressing = workerIdle, false
	a.render()
	if line := screenLine(a.screen, 0, 32); !strings.HasSuffix(line, "입력 0 · 출력 0") || strings.Contains(line, "MEM.") {
		t.Fatalf("narrow header overlapped existing counters: %q", line)
	}
}

func TestMemoryClockSkipsUnchangedIdleRepaints(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := screenFixture(t)
		screen := &memoryFrameScreen{Screen: a.screen}
		a.screen = screen
		var calls atomic.Int64
		a.memory.read = func() (uint64, error) { calls.Add(1); return 42 << 20, nil }
		done := make(chan error, 1)
		go func() { done <- a.loop() }()
		synctest.Wait()
		// Let startup's pending terminal frame settle before measuring idle ticks.
		time.Sleep(time.Second)
		synctest.Wait()
		initial := screen.frames.Load()
		initialSamples := calls.Load()
		time.Sleep(3 * time.Second)
		synctest.Wait()
		if calls.Load() != initialSamples+3 || screen.frames.Load() != initial {
			t.Fatalf("stable idle header: samples=%d frames=%d initial=%d", calls.Load(), screen.frames.Load(), initial)
		}
		screen.EventQ() <- tcell.NewEventKey(tcell.KeyCtrlC, "", 0)
		synctest.Wait()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	})
}

func TestMemoryReadFailureReplacesStaleValue(t *testing.T) {
	a := screenFixture(t)
	a.memory.read = func() (uint64, error) { return 42 << 20, nil }
	if !a.memory.sample() || a.memory.sample() {
		t.Fatal("unchanged sample requested another repaint")
	}
	a.memory.read = func() (uint64, error) { return 0, errors.New("unavailable") }
	if !a.memory.sample() || a.memory.sample() {
		t.Fatal("unavailable sample did not replace the previous value once")
	}
	a.render()
	if line := screenLine(a.screen, 0, 80); !strings.Contains(line, "MEM. USAGE —") || strings.Contains(line, "42.0 MiB") {
		t.Fatal(line)
	}
}

func TestMemoryRoundedChangesReuseLabel(t *testing.T) {
	for _, base := range []uint64{42 << 20, 1 << 30} {
		memory := memoryUsage{read: func() (uint64, error) { return base, nil }}
		memory.sample()
		var page uint64
		memory.read = func() (uint64, error) { page ^= 4096; return base + page, nil }
		if memory.sample() {
			t.Fatal("a page change within display precision requested repaint")
		}
		if allocs := testing.AllocsPerRun(100, func() { memory.sample() }); allocs != 0 {
			t.Fatalf("rounded unchanged sample allocated: %v", allocs)
		}
	}
}

func BenchmarkUnchangedMemorySample(b *testing.B) {
	memory := memoryUsage{read: func() (uint64, error) { return 42 << 20, nil }}
	memory.sample()
	b.ReportAllocs()
	for b.Loop() {
		memory.sample()
	}
}

func BenchmarkRoundedMemorySample(b *testing.B) {
	var page uint64
	memory := memoryUsage{read: func() (uint64, error) {
		page ^= 4096
		return 42<<20 + page, nil
	}}
	memory.sample()
	b.ReportAllocs()
	for b.Loop() {
		memory.sample()
	}
}
