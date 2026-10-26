// Purpose: Verify catalog discovery and model saving share one screen and never generate a diagnostic implicitly.
// Pattern: Local HTTP and terminal integration tests.
// Usage: go test ./internal/tui/session -run TestModelHints.
// Related: internal/tui/session/model_form_setup.go, internal/provider/setup/setup.go.
package session

import (
	"context"
	"fmt"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestModelHintsConnection(t *testing.T) {
	for _, mode := range []string{"candidate", "custom", "empty", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			var posts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					posts.Add(1)
					t.Error("settings generated a model response")
				}
				w.Header().Set("Content-Type", "application/json")
				if mode == "unavailable" {
					http.Error(w, "catalog unavailable", http.StatusNotFound)
					return
				}
				if mode == "empty" {
					fmt.Fprint(w, `{"data":[]}`)
					return
				}
				fmt.Fprint(w, `{"data":[{"id":"alpha"},{"id":"new-model"}]}`)
			}))
			defer server.Close()
			a := fixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			a.ctx = ctx
			defer func() { cancel(); a.shutdown() }()
			a.openModel()
			f := a.modelForm
			for i, p := range connection.Providers() {
				if p.Kind == connection.OpenRouter {
					f.provider = i
				}
			}
			f.endpoint.Set(server.URL)
			f.row = modelName
			a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
			for a.worker.active() {
				select {
				case n := <-a.events:
					a.receive(n)
				case <-a.ctx.Done():
					t.Fatal(a.ctx.Err())
				}
			}
			if a.page != "model" || !f.editing || posts.Load() != 0 {
				t.Fatal("catalog changed screens or generated a response")
			}
			expected := "new-model"
			if mode != "candidate" {
				expected = "custom/model"
			}
			query := expected
			if mode == "candidate" {
				query = "NEW"
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyRune, query, 0))
			if mode == "candidate" {
				a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
			if !f.confirm || a.selection.Model != "" {
				t.Fatal("confirmation was bypassed")
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
			drainTask(t, a, false)
			if a.selection.Model != expected || a.client == nil || a.page != "chat" || posts.Load() != 0 {
				t.Fatalf("selection=%+v status=%s", a.selection, a.status)
			}
		})
	}
}

func TestModelHintsRefreshOnReentry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Error("catalog lookup generated a model response")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"data":[{"id":"model-%d"}]}`, calls.Add(1))
	}))
	defer server.Close()
	a := fixture(t)
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()
	a.ctx = ctx
	a.openModel()
	f := a.modelForm
	for i, p := range connection.Providers() {
		if p.Kind == connection.OpenRouter {
			f.provider = i
		}
	}
	f.endpoint.Set(server.URL)
	f.row = modelName
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	drainTask(t, a, false)
	if calls.Load() != 1 || len(f.hints) != 1 || f.hints[0] != "model-1" {
		t.Fatalf("initial catalog: calls=%d hints=%v", calls.Load(), f.hints)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "model", 0))
	if calls.Load() != 1 || a.worker.active() {
		t.Fatal("typing fetched the catalog again")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	drainTask(t, a, false)
	if calls.Load() != 2 || f.name.String() != "model" || len(f.hints) != 1 || f.hints[0] != "model-2" {
		t.Fatalf("refreshed catalog: calls=%d input=%q hints=%v", calls.Load(), f.name.String(), f.hints)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if f.name.String() != "model-2" || f.editing || a.selection.Model != "" || a.client != nil {
		t.Fatal("refreshed selection did not remain an unsaved draft")
	}
}
