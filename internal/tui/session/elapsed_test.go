// Purpose: Verify shared-clock elapsed time advances monotonically and freezes on stop or completion.
// Pattern: Lifecycle tests with virtual time.
// Usage: go test ./internal/tui/session -run TestElapsed.
// Related: internal/tui/session/elapsed.go, internal/tui/session/app.go.
package session

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestElapsedClockLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := &app{work: workerActivity{state: workerSetup}}
		a.startElapsed()
		if a.activity().Label() != "상태 작업 중 00 : 00 : 00" {
			t.Fatal(a.activity().Label())
		}
		first := a.work.elapsedStarted
		time.Sleep(time.Hour + time.Minute + time.Second)
		a.updateElapsed(time.Now())
		if a.activity().Label() != "상태 작업 중 01 : 01 : 01" {
			t.Fatal(a.activity().Label())
		}
		a.stopElapsed()
		frozen := a.work.elapsed
		time.Sleep(2 * time.Second)
		a.updateElapsed(time.Now())
		if a.work.elapsed != frozen || !a.work.elapsedStarted.IsZero() {
			t.Fatal("stop did not freeze elapsed time")
		}
		a.startElapsed()
		if a.work.elapsed != 0 || a.work.elapsedStarted == first {
			t.Fatal("new task reused old duration")
		}
		a.requestExit()
		if !a.work.elapsedStarted.IsZero() {
			t.Fatal("exit retained active timing")
		}
	})
}

func TestElapsedWorkerCompletion(t *testing.T) {
	a := screenFixture(t)
	a.work.state = workerSetup
	a.startElapsed()
	a.receive(notice{Done: true})
	if a.work.state.active() || !a.work.elapsedStarted.IsZero() || a.activity().Label() != "상태 대기" {
		t.Fatal("worker completion retained active timing")
	}
}
