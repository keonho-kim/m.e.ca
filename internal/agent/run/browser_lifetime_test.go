// Purpose: Join the browser owner before final persistence even when the request is canceled.
// Pattern: Runtime resource lifetime regression.
// Usage: go test ./internal/agent/run -run TestBrowserLifetime.
// Related: internal/agent/run/conversation.go, internal/tools/local/browser.go.
package run

import (
	"context"
	"testing"
)

type browserExecutor struct {
	executor
	closed   bool
	canceled bool
	bounded  bool
}

func (e *browserExecutor) EndObservations(ctx context.Context) error {
	e.closed = true
	e.canceled = ctx.Err() != nil
	_, e.bounded = ctx.Deadline()
	return nil
}
func TestBrowserLifetime(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		tools := &browserExecutor{}
		g := &generator{replies: []string{"Finished"}}
		r := fixture(g, &tools.executor, &journal{})
		r.Tools = tools
		ctx, cancel := context.WithCancel(t.Context())
		if cancelled {
			r.PollSteering = func() []string { cancel(); return nil }
		}
		_, _ = r.Run(ctx, "Work", approve, emit)
		cancel()
		if !tools.closed || tools.canceled || !tools.bounded {
			t.Fatalf("invalid browser cleanup: %+v", tools)
		}
	}
}
