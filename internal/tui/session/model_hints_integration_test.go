// Purpose: Verify discovered candidates and custom IDs connect only after field confirmation.
// Pattern: HTTP and terminal workflow integration tests.
// Usage: go test ./internal/tui/session -run TestModelHintsConnection.
// Related: internal/tui/session/model.go, internal/tui/session/model_fields.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestModelHintsConnection(t *testing.T) {
	for _, mode := range []string{"candidate", "custom", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			custom := mode != "candidate"
			var posts atomic.Int32
			expected := "new-model"
			if custom {
				expected = "custom/model"
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					if mode == "unavailable" {
						http.Error(w, "model catalog unsupported", http.StatusNotFound)
						return
					}
					if custom {
						fmt.Fprint(w, `{"data":[]}`)
					} else {
						fmt.Fprint(w, `{"data":[{"id":"alpha"},{"id":"new-model"}]}`)
					}
					return
				}
				posts.Add(1)
				var body struct{ Model string }
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				if body.Model != expected {
					t.Errorf("wire model=%q", body.Model)
				}
				fmt.Fprint(w, `{"choices":[{"message":{"content":"Hello"},"finish_reason":"stop"}]}`)
			}))
			defer server.Close()
			a := fixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			a.ctx = ctx
			defer func() { cancel(); a.shutdown() }()
			conn, err := connection.New(connection.OpenAICompatible, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			a.connect(conn, "", "", false)
			for a.busy {
				select {
				case n := <-a.events:
					a.receive(n)
					if n.Err != nil && mode != "unavailable" {
						t.Fatal(n.Err)
					}
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			if a.page != "manual" || !a.modelHintsOpen || posts.Load() != 0 {
				t.Fatal("discovery did not open model field")
			}
			if mode == "unavailable" && a.status == "" {
				t.Fatal("discovery error hidden")
			}
			query := "NEW"
			if custom {
				query = expected
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyRune, query, 0))
			if !custom {
				a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
				if a.busy || posts.Load() != 0 || a.menuInput.String() != expected {
					t.Fatal("candidate selection connected or lost ID")
				}
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
			drainTask(t, a, false)
			if a.selection.Model != expected || a.client == nil || posts.Load() != 1 {
				t.Fatalf("selected=%q posts=%d", a.selection.Model, posts.Load())
			}
		})
	}
}
