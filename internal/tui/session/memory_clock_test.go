// Purpose: Verify adaptive memory sampling preserves command cadence and releases the shared clock.
// Pattern: Event-loop behavior tests with virtual time and atomic observations.
// Usage: go test ./internal/tui/session -run TestMemoryClock.
// Related: internal/tui/session/event_loop.go, internal/tui/session/memory_usage.go, internal/tui/session/command_activity.go.
package session

import (
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gdamore/tcell/v3"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type memoryCommandTools struct {
	Tools
	toolapi.CommandExecutor
	calls atomic.Int64
}

func (t *memoryCommandTools) CommandActivity() []toolapi.CommandRun {
	t.calls.Add(1)
	return nil
}

func TestMemoryClockSamplesIdleAndActive(t *testing.T) {
	for _, busy := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			a := screenFixture(t)
			var calls atomic.Int64
			a.memory.read = func() (uint64, error) { return uint64(calls.Add(1)) << 20, nil }
			commands := &memoryCommandTools{}
			a.opts.Tools = commands
			interval := time.Second
			if busy {
				interval = 250 * time.Millisecond
				a.work.state = workerSetup
				a.startElapsed()
			}
			done := make(chan error, 1)
			go func() { done <- a.loop() }()
			synctest.Wait()
			if calls.Load() != 1 {
				t.Fatalf("initial samples=%d", calls.Load())
			}
			time.Sleep(interval - time.Millisecond)
			synctest.Wait()
			if calls.Load() != 1 {
				t.Fatal("sampled before the interval elapsed")
			}
			time.Sleep(time.Millisecond)
			synctest.Wait()
			if calls.Load() != 2 {
				t.Fatalf("samples=%d", calls.Load())
			}
			if busy {
				if commands.calls.Load() != 0 {
					t.Fatal("memory sample queried command metadata within the same second")
				}
				time.Sleep(time.Second - interval)
				synctest.Wait()
				if calls.Load() != 5 || a.work.elapsed != time.Second || commands.calls.Load() != 1 {
					t.Fatalf("samples=%d elapsed=%v command reads=%d", calls.Load(), a.work.elapsed, commands.calls.Load())
				}
			} else if commands.calls.Load() != 0 {
				t.Fatal("idle memory sample queried command metadata")
			}
			a.screen.EventQ() <- tcell.NewEventKey(tcell.KeyCtrlC, "", 0)
			synctest.Wait()
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			stopped := calls.Load()
			time.Sleep(2 * time.Second)
			synctest.Wait()
			if calls.Load() != stopped {
				t.Fatal("sampling continued after the event loop closed")
			}
		})
	}
}

func TestMemoryClockRevertsToIdleCadence(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := screenFixture(t)
		var calls atomic.Int64
		a.memory.read = func() (uint64, error) { calls.Add(1); return 42 << 20, nil }
		a.work.state = workerSetup
		a.startElapsed()
		done := make(chan error, 1)
		go func() { done <- a.loop() }()
		synctest.Wait()
		time.Sleep(250 * time.Millisecond)
		synctest.Wait()
		if calls.Load() != 2 {
			t.Fatalf("active samples=%d", calls.Load())
		}
		a.events <- notice{Done: true}
		synctest.Wait()
		time.Sleep(999 * time.Millisecond)
		synctest.Wait()
		if calls.Load() != 2 {
			t.Fatal("completed worker retained active sampling cadence")
		}
		time.Sleep(time.Millisecond)
		synctest.Wait()
		if calls.Load() != 3 {
			t.Fatal("idle sampling did not resume after completion")
		}
		a.screen.EventQ() <- tcell.NewEventKey(tcell.KeyCtrlC, "", 0)
		synctest.Wait()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	})
}

func TestMemoryClockPreservesPendingStreamFrame(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := screenFixture(t)
		var calls atomic.Int64
		a.memory.read = func() (uint64, error) { return uint64(calls.Add(1)) << 20, nil }
		a.work.state = workerSetup
		a.startElapsed()
		done := make(chan error, 1)
		go func() { done <- a.loop() }()
		synctest.Wait()
		// These virtual offsets place the memory tick inside a coalesced stream frame.
		time.Sleep(245 * time.Millisecond)
		synctest.Wait()
		a.events <- notice{Event: &runrecord.Event{Kind: "stream", Text: "first text"}}
		synctest.Wait()
		time.Sleep(4 * time.Millisecond)
		a.events <- notice{Event: &runrecord.Event{Kind: "stream", Text: "latest text"}}
		synctest.Wait()
		time.Sleep(21 * time.Millisecond)
		synctest.Wait()
		if got := screenLine(a.screen, 1, 80); !strings.Contains(got, "latest text") {
			t.Fatalf("memory-only refresh dropped the pending body update: %q", got)
		}
		if calls.Load() != 2 {
			t.Fatal("the pending frame did not overlap a memory sample")
		}
		a.screen.EventQ() <- tcell.NewEventKey(tcell.KeyCtrlC, "", 0)
		synctest.Wait()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	})
}
