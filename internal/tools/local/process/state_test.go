// Purpose: Verify command elapsed time and actual output age remain distinct observations.
// Pattern: Deterministic clock and capture contract tests.
// Usage: go test -race ./internal/tools/local/process -run TestCommandSnapshotOutputTiming.
// Related: internal/tools/local/process/state.go, internal/tools/local/process/output.go.
package process

import (
	"encoding/json"
	"testing"
	"testing/synctest"
	"time"
)

func TestCommandSnapshotOutputTiming(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		j := &job{state: State{Status: "running"}, started: time.Now(), done: make(chan struct{})}
		check := func(elapsed int64, age *int64) {
			t.Helper()
			body, err := json.Marshal(j.snapshot(false))
			if err != nil {
				t.Fatal(err)
			}
			var observed struct {
				Elapsed int64  `json:"elapsed_millis"`
				Age     *int64 `json:"last_output_millis"`
			}
			if err := json.Unmarshal(body, &observed); err != nil {
				t.Fatal(err)
			}
			if observed.Elapsed != elapsed || (age == nil) != (observed.Age == nil) || age != nil && *observed.Age != *age {
				t.Fatalf("elapsed=%d: expected last output %v, got %s", elapsed, age, body)
			}
		}
		time.Sleep(5 * time.Second)
		check(5000, nil)
		j.stdout.Write([]byte("first output"))
		time.Sleep(3 * time.Second)
		age := int64(3000)
		check(8000, &age)
		j.stderr.Write([]byte("new diagnostic"))
		time.Sleep(2 * time.Second)
		age = 2000
		check(10000, &age)
		j.stdout.Write(nil)
		j.stderr.Write(nil)
		j.stdout.release()
		j.stderr.release()
		time.Sleep(time.Second)
		age = 3000
		check(11000, &age)
	})
}
