// Purpose: Drive task workers and approvals through the terminal test harness.
// Pattern: Integration behavior tests.
// Usage: go test ./internal/tui/session.
// Related: internal/tui/session/tasks.go, internal/provider/protocol/request.go, internal/storage/sqlite/events.go.
package session

import "testing"

func drainTask(t *testing.T, a *app, approve bool) {
	t.Helper()
	for a.worker.active() {
		select {
		case n := <-a.events:
			a.receive(n)
			if n.Approval != nil {
				if !approve {
					t.Fatal("unexpected mutation approval")
				}
				a.decide(true)
			}
			if n.Done && n.Err != nil {
				t.Fatal(n.Err)
			}
		case <-a.ctx.Done():
			t.Fatal(a.ctx.Err())
		}
	}
}
