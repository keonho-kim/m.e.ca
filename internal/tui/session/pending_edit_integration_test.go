// Purpose: Verify task completion waits for queued edits before ordered model execution and persistence.
// Pattern: Controlled HTTP and temporary-session integration tests.
// Usage: go test ./internal/tui/session -run TestPendingEditCompletionWaits.
// Related: internal/tui/session/pending_edit.go, internal/tui/session/pending.go, internal/tui/session/worker_result.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
)

func TestPendingEditCompletionWaits(t *testing.T) {
	for _, decision := range []string{"save", "cancel", "delete"} {
		t.Run(decision, func(t *testing.T) {
			a := fixture(t)
			ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
			a.ctx = ctx
			started, release := make(chan struct{}), make(chan struct{})
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Messages []modelapi.Message `json:"messages"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				attempt := calls.Add(1)
				if attempt == 1 {
					close(started)
					select {
					case <-release:
					case <-ctx.Done():
						return
					}
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"choices":[{"message":{"content":"answer %d"},"finish_reason":"stop"}]}`, attempt)
			}))
			t.Cleanup(func() { cancel(); a.shutdown(); server.Close() })
			c, err := connection.New(connection.OpenRouter, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			a.client, err = httpclient.NewClient(c, "")
			if err != nil {
				t.Fatal(err)
			}
			a.selection = connection.Selection{Connection: c, Model: "mock"}
			a.startTask("current request")
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			for _, text := range []string{"first queued", "second queued"} {
				a.editor.Set(text)
				a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
			}
			a.editor.Set("unsent draft")
			a.editor.MoveTo(3)
			pendingArrow(a, tcell.KeyLeft)
			a.keyEvent(tcell.NewEventKey(tcell.KeyRune, " revised", 0))
			close(release)
			drainTask(t, a, false)
			if calls.Load() != 1 || a.queued.Edit == nil || a.editor.String() != "second queued revised" || len(a.queued.Messages) != 2 {
				t.Fatal("task completion consumed queued input while it was being edited")
			}
			key := map[string]tcell.Key{"save": tcell.KeyEnter, "cancel": tcell.KeyEscape, "delete": tcell.KeyBackspace}[decision]
			modifiers := tcell.ModNone
			if decision == "delete" {
				modifiers = tcell.ModCtrl
			}
			a.keyEvent(tcell.NewEventKey(key, "", modifiers))
			drainTask(t, a, false)
			want := []string{"current request", "first queued"}
			if decision == "save" {
				want = append(want, "second queued revised")
			} else if decision == "cancel" {
				want = append(want, "second queued")
			}
			history, err := a.opts.Store.Messages(a.ctx, a.opts.Session.ID, 100)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, m := range history {
				if m.Role == "user" {
					got = append(got, m.Content)
				}
			}
			if !slices.Equal(got, want) || calls.Load() != int32(len(want)) || a.queued.Messages != nil ||
				a.queued.Edit != nil || a.editor.String() != "unsent draft" || a.editor.Cursor != 3 {
				t.Fatalf("queued history=%v; want %v, input=%q cursor=%d", got, want, a.editor.String(), a.editor.Cursor)
			}
		})
	}
}
