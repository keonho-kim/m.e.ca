// Purpose: Verify successful model selection is reused by new sessions while existing session choices remain authoritative.
// Pattern: Model connection and persistence integration test.
// Usage: go test ./internal/tui/session -run TestSavedModel.
// Related: internal/tui/session/connection.go, internal/tui/session/session.go, internal/config/model.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/provider/connection"
)

func TestSavedModelReusedWithSessionPrecedence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `{"data":[{"id":"test-model"},{"id":"session-model"}]}`)
			return
		}
		var request struct{ Messages []json.RawMessage }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		reply := `{"tool":"read_file"}`
		if len(request.Messages) > 2 {
			reply = "AGENTS.md"
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": reply}, "finish_reason": "stop"}}}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	a := fixture(t)
	a.ctx = ctx
	t.Cleanup(a.shutdown)
	conn, err := connection.New(connection.OpenAICompatible, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	a.connect(conn, "", "test-model", true)
	drainTask(t, a, false)
	cfg, err := config.Load(a.opts.State)
	if err != nil || cfg.Model == nil || cfg.Model.Name != "test-model" {
		t.Fatalf("default was not saved: %+v %v", cfg.Model, err)
	}
	for _, ownModel := range []string{"", "session-model"} {
		t.Run(ownModel, func(t *testing.T) {
			b := fixture(t)
			b.ctx, b.opts.Config = ctx, cfg
			t.Cleanup(b.shutdown)
			if ownModel != "" {
				if err := b.opts.Store.SaveSelection(ctx, b.opts.Session.ID, connection.Selection{Connection: conn, Model: ownModel}); err != nil {
					t.Fatal(err)
				}
			}
			b.opts.SaveConfig = func(config.Config) error { t.Error("restoration overwrote global default"); return nil }
			if err := b.restore(); err != nil {
				t.Fatal(err)
			}
			drainTask(t, b, false)
			want := ownModel
			if want == "" {
				want = "test-model"
			}
			selected, err := b.opts.Store.Selection(ctx, b.opts.Session.ID)
			if err != nil || selected.Model != want || b.client == nil || b.selection.Model != want {
				t.Fatalf("restored %+v %v", selected, err)
			}
		})
	}
}
