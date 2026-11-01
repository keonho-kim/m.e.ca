// Purpose: Verify unrestricted FIFO admission and ordered draft recovery after interruption or dispatch failure.
// Pattern: Terminal state transition tests.
// Usage: go test ./internal/tui/session -run TestPending.
// Related: internal/tui/session/pending.go, internal/tui/session/keys.go.
package session

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestPendingAcceptsManyRequests(t *testing.T) {
	a := screenFixture(t)
	a.work.state = workerAgent
	var requests []string
	for i := range 1024 {
		text := fmt.Sprintf("request %d", i)
		requests = append(requests, text)
		a.compose.editor.Set(text)
		a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
		if a.compose.editor.String() != "" || !a.work.state.agent() {
			t.Fatalf("request %d was not reserved", i)
		}
	}
	if keymapLabels(t, a)["Tab"] != "후속 예약" {
		t.Fatal("Tab hint disappeared while requests were queued")
	}
	a.restorePending()
	if a.compose.editor.String() != strings.Join(requests, "\n\n") {
		t.Fatal("restoration lost or reordered queued requests")
	}
}

func TestPendingInputPreservesOrderAndDraft(t *testing.T) {
	a := screenFixture(t)
	a.work.state = workerAgent
	a.compose.editor.Set("next request")
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if len(a.compose.queued.Messages) != 1 || a.compose.queued.Messages[0].Text != "next request" || a.compose.editor.String() != "" || !a.work.state.active() {
		t.Fatal("Tab did not reserve input")
	}
	a.compose.editor.Set("later draft")
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if len(a.compose.queued.Messages) != 2 || a.compose.queued.Messages[0].Text != "next request" || a.compose.queued.Messages[1].Text != "later draft" || a.compose.editor.String() != "" {
		t.Fatal("second Tab did not append the request")
	}
	a.compose.editor.Set("unsent draft")
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()
	a.work.cancel = cancel
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if ctx.Err() != context.Canceled || len(a.compose.queued.Messages) != 0 || a.compose.editor.String() != "next request\n\nlater draft\n\nunsent draft" {
		t.Fatal("Esc lost pending input")
	}
}

func TestPendingRejectsLocalCommands(t *testing.T) {
	a := screenFixture(t)
	a.work.state = workerAgent
	a.compose.editor.Set("!quit")
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if len(a.compose.queued.Messages) != 0 || a.exit || a.compose.editor.String() != "!quit" {
		t.Fatal("local command was queued or executed")
	}
}

func TestPendingFailureRestoresDraftWithoutStarting(t *testing.T) {
	a := screenFixture(t)
	a.work.state = workerAgent
	a.compose.queued.Messages = queuedRequests("next request", "later request")
	a.compose.editor.Set("unsent draft")
	a.receive(notice{Done: true, Err: context.DeadlineExceeded})
	if a.work.state.active() || len(a.compose.queued.Messages) != 0 || a.compose.editor.String() != "next request\n\nlater request\n\nunsent draft" {
		t.Fatal("failed worker started or lost pending request")
	}
}

func TestBlockedCompletionRestoresReservedInput(t *testing.T) {
	a := screenFixture(t)
	a.work.state = workerAgent
	a.compose.queued.Messages = queuedRequests("next request", "later request")
	a.receive(notice{Done: true, RunStatus: runrecord.Stopped})
	if len(a.compose.queued.Messages) != 0 || a.compose.editor.String() != "next request\n\nlater request" || a.work.state.active() {
		t.Fatal("blocked completion consumed the reserved request")
	}
}

func TestPendingDispatchFailureRestoresWholeQueue(t *testing.T) {
	a := screenFixture(t)
	a.compose.queued.Messages = queuedRequests("first", "second", "third")
	retained := a.compose.queued.Messages
	a.compose.editor.Set("unsent draft")
	a.finishPending(true)
	if a.work.state.active() || a.compose.queued.Messages != nil || a.compose.editor.String() != "first\n\nsecond\n\nthird\n\nunsent draft" {
		t.Fatal("failed dispatch lost or started queued requests")
	}
	for _, text := range retained {
		if text.Text != "" || text.Paths != nil {
			t.Fatal("restored queue retained request references")
		}
	}
}

func TestPendingProjectionShowsCountAndEveryRequest(t *testing.T) {
	for _, language := range []locale.Language{locale.Korean, locale.English} {
		a := screenFixture(t)
		a.language = language
		a.compose.queued.Messages = queuedRequests("first", "second", "third")
		transcript := a.chatLayout(80)
		rows := transcript.Window(0, transcript.Count())
		var lines []string
		for _, row := range rows {
			lines = append(lines, row.Text)
		}
		want := language.Format("Queued (%d)", 3) + "\n  1. first\n  2. second\n  3. third"
		if got := strings.Join(lines, "\n"); got != want {
			t.Fatalf("queue projection=%q; want %q", got, want)
		}
		a.restorePending()
		a.chatLayout(80)
		if a.chat.chatProjection.queue.Header.Text != "" {
			t.Fatal("restored queue remained visible")
		}
	}
}
