// Purpose: Establish an idle resumed SSE connection before waiting for the next engine event.
// Pattern: Streaming handshake regression test.
// Usage: go test ./internal/serverapi -run TestSSEIdleResume.
// Related: internal/serverapi/http.go, internal/commands/server/events.go.
package serverapi

import (
	"testing"

	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func TestSSEIdleResumeEstablishesBeforeNextEvent(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	w := tlsWire(t, "sse", f.api)
	ctx := deadline(t)
	watch, err := w.watch(ctx, f.app.State().Cursor)
	if err != nil {
		t.Fatalf("SSE handshake waited for an engine event: %v", err)
	}
	defer watch.close()
	call(t, ctx, w, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}})
	if n := next(t, watch); n.GetEvent().GetSessionId() == "" || n.GetEvent().GetKind() != "session_opened" {
		t.Fatal("resumed observer lost the session transition", n)
	}
}
