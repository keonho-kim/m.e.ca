// Purpose: Verify elapsed time advances from monotonic time and releases its ticker on stop or restart.
// Pattern: Lifecycle tests with virtual time.
// Usage: go test ./internal/tui/session -run TestElapsed.
// Related: internal/tui/session/elapsed.go, internal/tui/session/app.go.
package session

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestElapsedTimerLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := &app{work: workerActivity{state: workerSetup}}
		a.startElapsed()
		if a.activity().Label() != "상태 작업 중 00 : 00 : 00" {
			t.Fatal(a.activity().Label())
		}
		first := a.work.elapsedTimer
		time.Sleep(time.Hour + time.Minute + time.Second)
		<-a.work.elapsedTicks
		a.updateElapsed(time.Now())
		if a.activity().Label() != "상태 작업 중 01 : 01 : 01" {
			t.Fatal(a.activity().Label())
		}
		a.stopElapsed()
		frozen := a.work.elapsed
		time.Sleep(2 * time.Second)
		a.updateElapsed(time.Now())
		if a.work.elapsed != frozen || a.work.elapsedTicks != nil || a.work.elapsedTimer != nil {
			t.Fatal("stop did not freeze/release timer")
		}
		a.startElapsed()
		if a.work.elapsed != 0 || a.work.elapsedTimer == first {
			t.Fatal("new task reused old duration")
		}
		a.requestExit()
		if a.work.elapsedTicks != nil || a.work.elapsedTimer != nil {
			t.Fatal("exit leaked ticker")
		}
	})
}

func TestElapsedWorkerCompletion(t *testing.T) {
	a := screenFixture(t)
	a.work.state = workerSetup
	a.startElapsed()
	a.receive(notice{Done: true})
	if a.work.state.active() || a.work.elapsedTicks != nil || a.activity().Label() != "상태 대기" {
		t.Fatal("worker completion retained elapsed timer")
	}
}
