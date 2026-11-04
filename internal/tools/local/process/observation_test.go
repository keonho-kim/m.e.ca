// Purpose: Keep command timing observable while capture storage is occupied.
// Pattern: Deterministic synchronization boundary test.
// Usage: Process tests exercise metadata reads independently of output I/O.
// Related: internal/tools/local/process/capture.go, internal/tools/local/process/state.go.
package process

import (
	"testing"
	"time"
)

func TestCommandTimingDoesNotWaitForCaptureIO(t *testing.T) {
	j := &job{started: time.Now(), done: make(chan struct{})}
	j.stdout.Write([]byte("observed"))
	j.stdout.mu.Lock()
	ready := make(chan State, 1)
	go func() { ready <- j.snapshot() }()
	select {
	case state := <-ready:
		j.stdout.mu.Unlock()
		if state.LastOutputMillis == nil {
			t.Fatal("actual output timing lost")
		}
	case <-time.After(2 * time.Second):
		j.stdout.mu.Unlock()
		<-ready
		t.Fatal("metadata observation waited for capture I/O")
	}
}
