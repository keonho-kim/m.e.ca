// Purpose: Verify effort menus are model-specific and failed persistence preserves the active preference.
// Pattern: Terminal settings behavior tests.
// Usage: go test ./internal/tui/session -run TestEffort.
// Related: internal/tui/session/effort.go, internal/tui/session/menus.go, internal/tui/view/header.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gdamore/tcell/v3"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEffortMenuAndOffHeader(t *testing.T) {
	a := fixture(t)
	connection, err := modelconn.New(modelconn.VLLM, "")
	if err != nil {
		t.Fatal(err)
	}
	medium := "medium"
	on := true
	a.selection = modelconn.Selection{Connection: connection, Model: "Qwen/Qwen3.8-27B", Profile: "qwen3.8-27b", Thinking: &on, ThinkingSupported: true, ReasoningEffort: &medium}
	a.openModel()
	a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "effort" || a.choice != 2 || !reflect.DeepEqual(a.choices, []string{"DEFAULT", "LOW", "MEDIUM", "XHIGH"}) {
		t.Fatalf("effort menu=%s %v index=%d", a.page, a.choices, a.choice)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	a.render()
	if got := screenLine(a.screen, 1, 80); !strings.Contains(got, "Thinking MEDIUM") {
		t.Fatalf("strength hidden: %q", got)
	}
	off := false
	a.selection.Thinking = &off
	a.render()
	if got := screenLine(a.screen, 1, 80); strings.Contains(got, "Thinking") {
		t.Fatalf("OFF still displays strength: %q", got)
	}
	if a.selection.Effort() != "medium" {
		t.Fatal("turning off erased strength preference")
	}
}

func TestEffortUnsupportedModelMenuCannotOpen(t *testing.T) {
	a := fixture(t)
	connection, err := modelconn.New(modelconn.VLLM, "")
	if err != nil {
		t.Fatal(err)
	}
	a.selection = modelconn.Selection{Connection: connection, Model: "Ornith-1.5-9B", Profile: "ornith-1.5-9b", ThinkingSupported: true}
	a.openModel()
	a.choice = 2
	a.choose()
	if a.page != "model" || a.status == "" {
		t.Fatal("unverified derivative effort menu opened")
	}
}

func TestEffortPersistenceFailurePreservesPreference(t *testing.T) {
	a := fixture(t)
	previous := "low"
	connection, err := modelconn.New(modelconn.VLLM, "")
	if err != nil {
		t.Fatal(err)
	}
	a.selection = modelconn.Selection{Model: "Qwen/Qwen3.8-27B", Profile: "qwen3.8-27b", Connection: connection, ThinkingSupported: true}
	a.selection.ReasoningEffort = &previous
	if err := closeTestStore(t, a.opts.Store); err != nil {
		t.Fatal(err)
	}
	next := "medium"
	a.receive(notice{Done: true, Effort: &next})
	if a.selection.Effort() != "low" || a.status == "" {
		t.Fatal("save failure silently changed preference")
	}
}

func TestEffortAsyncApplicationAndCancellation(t *testing.T) {
	for _, cancelProbe := range []bool{false, true} {
		t.Run(map[bool]string{false: "apply", true: "cancel"}[cancelProbe], func(t *testing.T) {
			a := fixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			a.ctx = ctx
			t.Cleanup(func() {
				cancel()
				a.workers.Wait()
				if a.client != nil {
					a.client.Close()
				}
			})
			started := make(chan struct{}, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Effort string `json:"reasoning_effort"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				if body.Effort != "medium" {
					t.Errorf("effort=%s", body.Effort)
				}
				started <- struct{}{}
				if cancelProbe {
					<-r.Context().Done()
					return
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"choices":[{"message":{"content":"391","reasoning":"17 times 23 is 391"},"finish_reason":"stop"}]}`)
			}))
			t.Cleanup(server.Close)
			connection, err := modelconn.New(modelconn.VLLM, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			a.client, err = httpclient.NewClient(connection, "", "")
			if err != nil {
				t.Fatal(err)
			}
			low := "low"
			off := false
			a.selection = modelconn.Selection{Connection: connection, Model: "Qwen/Qwen3.8-27B", Profile: "qwen3.8-27b", Thinking: &off, ThinkingSupported: true, ReasoningEffort: &low}
			a.openEffort()
			a.choice = 2
			a.chooseEffort()
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			if cancelProbe {
				a.taskCancel()
			}
			select {
			case n := <-a.events:
				a.receive(n)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			want := "medium"
			if cancelProbe {
				want = "low"
			}
			if a.selection.Effort() != want || *a.selection.Thinking || a.busy {
				t.Fatalf("preference changed incorrectly: %+v busy=%t", a.selection, a.busy)
			}
			if !cancelProbe {
				stored, err := a.opts.Store.Selection(ctx, a.opts.Session.ID)
				if err != nil || stored.Effort() != want {
					t.Fatalf("effort not saved: %+v %v", stored, err)
				}
			} else if a.status == "" {
				t.Fatal("cancelled probe reported success")
			}
		})
	}
}
