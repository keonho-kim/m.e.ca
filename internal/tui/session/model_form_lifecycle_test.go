// Purpose: Verify model-form cancellation, stale query rejection and secret-safe editing.
// Pattern: Virtual terminal and cancellable local HTTP tests.
// Usage: go test ./internal/tui/session -run TestModelForm.
// Related: internal/tui/session/model_form_setup.go, internal/tui/session/render_model.go, internal/tui/session/paste.go.
package session

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
)

func TestModelFormMasksKeyAndPastesWithoutSaving(t *testing.T) {
	a := fixture(t)
	a.openModel()
	f := a.modelForm
	f.Row = modelsettings.Credential
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	a.pasteBoundary(true)
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "!quit-private-key", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.pasteBoundary(false)
	body := modelScreen(a)
	if a.exit || f.Confirm || a.worker.active() || !strings.Contains(body, "****") || strings.Contains(body, "!quit-private-key") || a.inputRegion.editor != nil {
		t.Fatal("credential exposed or paste submitted")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "model" || f.Credential.String() != "!quit-private-key" {
		t.Fatal("confirmation Esc discarded the draft")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "N", 0))
	if a.page != "chat" || a.modelForm != nil || f.Credential.String() != "" {
		t.Fatal("discard retained secret input")
	}
}

func TestModelFormCancelLookupKeepsDraft(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }))
	defer server.Close()
	a := fixture(t)
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()
	a.ctx = ctx
	a.openModel()
	f := a.modelForm
	f.Endpoint.Set(server.URL)
	f.Row = modelsettings.Name
	f.Name.Set("custom/model")
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	for a.worker.active() {
		select {
		case n := <-a.events:
			a.receive(n)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	if a.page != "model" || a.modelForm != f || f.Endpoint.String() != server.URL || f.Name.String() != "custom/model" || a.client != nil {
		t.Fatal("cancelled lookup changed the active connection or draft")
	}
}

func TestModelFormRejectsResultFromEarlierDraft(t *testing.T) {
	a := fixture(t)
	a.openModel()
	f := a.modelForm
	c, err := connection.New(connection.VLLM, "")
	if err != nil {
		t.Fatal(err)
	}
	client, err := httpclient.NewClient(c, "")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	previous := f.Revision
	f.Invalidate(true)
	next := connection.Selection{Connection: c, Model: "stale"}
	a.finishModelFormSetup(notice{FormRevision: previous, Client: client, Selection: &next, ModelForm: true})
	if f.Client != nil || f.Prepared != nil || a.selection.Model != "" || a.status == "" {
		t.Fatal("stale draft result was adopted")
	}
}

func TestModelFormSelectedCandidateVisibleAfterResize(t *testing.T) {
	a := fixture(t)
	a.openModel()
	f := a.modelForm
	f.Row, f.Editing = modelsettings.Name, true
	f.Models = []string{"one", "two", "three", "four", "five", "six", "seven", "eight"}
	f.RefreshHints()
	a.screen.SetSize(24, 10)
	for i := 0; i < 7; i++ {
		a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	}
	if body := modelScreen(a); !strings.Contains(body, "eight") {
		t.Fatalf("selected candidate hidden: %s", body)
	}
}

func TestModelFormAcceptsTypingAndSaveWhileCatalogLoads(t *testing.T) {
	for _, decision := range []string{"y", "N"} {
		t.Run(decision, func(t *testing.T) {
			started, release := make(chan struct{}), make(chan struct{})
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					close(started)
					select {
					case <-release:
					case <-r.Context().Done():
						return
					}
				}
				fmt.Fprint(w, `{"data":[{"id":"alias"}]}`)
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
					f.Provider = i
				}
			}
			f.Endpoint.Set(server.URL)
			f.Row = modelsettings.Name
			a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "alias", 0))
			if f.Name.String() != "alias" {
				t.Fatal("catalog lookup dropped typed input")
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
			a.keyEvent(tcell.NewEventKey(tcell.KeyRune, decision, 0))
			if a.page != "chat" {
				t.Fatal("save decision did not return to chat")
			}
			close(release)
			for a.worker.active() {
				select {
				case n := <-a.events:
					a.receive(n)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			if decision == "y" && (a.selection.Model != "alias" || a.client == nil || a.modelForm != nil) {
				t.Fatalf("queued save failed: %+v %s", a.selection, a.status)
			}
			if decision == "N" && (a.selection.Model != "" || a.client != nil || a.modelForm != nil) {
				t.Fatal("discarded catalog lookup committed settings")
			}
		})
	}
}
