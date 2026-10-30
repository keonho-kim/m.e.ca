// Purpose: Verify session transitions release old database ownership and reconnect the global model in a fresh conversation.
// Pattern: Composition and virtual terminal integration test.
// Usage: go test ./cmd/meca -run TestNewConversation.
// Related: cmd/meca/conversation.go, internal/config/model.go, internal/tui/session/app.go.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	tuisession "github.com/keonho-kim/meca/internal/tui/session"
)

func TestNewConversationReleasesLeaseAndRetainsGlobalModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `{"data":[{"id":"saved-model"}]}`)
			return
		}
		var body struct{ Messages []json.RawMessage }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		reply := "Hello!"
		if err := json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": reply}, "finish_reason": "stop"}}}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	home, project := t.TempDir(), t.TempDir()
	cfg, err := bootstrap.Initialize(home)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := connection.New(connection.OpenRouter, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Model = config.RememberModel(connection.Selection{Connection: conn, Model: "saved-model"})
	state := filepath.Join(home, ".meca")
	if err := config.Save(state, cfg); err != nil {
		t.Fatal(err)
	}
	sessions, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	screen, err := tcell.NewTerminfoScreenFromTty(vt.NewMockTerm(vt.MockOptSize{X: 100, Y: 24}))
	if err != nil {
		t.Fatal(err)
	}
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	previousID := ""
	for _, command := range []string{"!new", "!quit"} {
		store, err := sessions.Create(ctx, project, project)
		if err != nil {
			t.Fatal(err)
		}
		id := store.Session().ID
		if id == previousID {
			t.Fatal("session database was reused")
		}
		if command == "!new" {
			if err := store.AppendMessage(ctx, id, modelapi.Message{Role: "user", Content: "previous conversation"}); err != nil {
				t.Fatal(err)
			}
		}
		type result struct {
			action tuisession.Action
			err    error
		}
		finished := make(chan result, 1)
		go func() {
			action, err := runConversation(ctx, screen, cfg, home, state, store)
			finished <- result{action, err}
		}()
		ticker := time.NewTicker(10 * time.Millisecond)
		connected := false
		for !connected {
			select {
			case <-ctx.Done():
				ticker.Stop()
				<-finished
				t.Fatal("connection did not finish")
			case <-ticker.C:
				selected, err := store.Selection(ctx, id)
				if err != nil {
					cancel()
					ticker.Stop()
					<-finished
					t.Fatal(err)
				}
				connected = selected.Model == "saved-model"
			}
		}
		ticker.Stop()
		screen.EventQ() <- tcell.NewEventKey(tcell.KeyRune, command, 0)
		screen.EventQ() <- tcell.NewEventKey(tcell.KeyEnter, "", 0)
		got := <-finished
		want := tuisession.Exit
		if command == "!new" {
			want = tuisession.NewSession
		}
		if got.err != nil || got.action != want {
			t.Fatalf("%+v", got)
		}
		reopened, err := sessions.Open(ctx, project, id)
		if err != nil {
			t.Fatal(err)
		}
		lease, err := reopened.LeaseSession()
		if err != nil {
			t.Fatal(err)
		}
		messages, err := reopened.ContextMessages(ctx, id, 0, 16*1024*1024, true)
		if err != nil {
			t.Fatal(err)
		}
		if command == "!quit" && len(messages) != 0 {
			t.Fatal("new session inherited old conversation")
		}
		if err := lease.Close(); err != nil {
			t.Fatal(err)
		}
		if err := reopened.Close(); err != nil {
			t.Fatal(err)
		}
		previousID = id
		cfg, err = config.Load(state)
		if err != nil || cfg.Model == nil || cfg.Model.Name != "saved-model" {
			t.Fatal("global connection was lost")
		}
	}
}
