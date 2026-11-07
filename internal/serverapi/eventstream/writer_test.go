// Purpose: Verify idle heartbeat timing and activity resets without real-time waits.
// Pattern: Deterministic streaming contract test.
// Usage: go test ./internal/serverapi/eventstream.
// Related: internal/serverapi/eventstream/writer.go.
package eventstream

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestHeartbeatStartsAfterFiveSecondsAndRepeatsEachSecond(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		recorder := httptest.NewRecorder()
		writer, err := New(recorder)
		if err != nil {
			t.Fatal(err)
		}
		assertDelay := func(want time.Duration) {
			t.Helper()
			ctx, cancel := writer.PollContext(context.Background())
			defer cancel()
			deadline, _ := ctx.Deadline()
			if got := time.Until(deadline); got != want {
				t.Fatalf("heartbeat in %s, want %s", got, want)
			}
		}
		assertDelay(5 * time.Second)
		time.Sleep(5 * time.Second)
		if err := writer.Heartbeat(); err != nil {
			t.Fatal(err)
		}
		assertDelay(time.Second)
		time.Sleep(time.Second)
		if err := writer.Heartbeat(); err != nil {
			t.Fatal(err)
		}
		assertDelay(time.Second)
		if err := writer.Send("", "", []byte(`{"text":"hello"}`)); err != nil {
			t.Fatal(err)
		}
		assertDelay(5 * time.Second)
		if got := strings.Count(recorder.Body.String(), ": heartbeat\n\n"); got != 2 {
			t.Fatal(recorder.Body.String())
		}
	})
}
