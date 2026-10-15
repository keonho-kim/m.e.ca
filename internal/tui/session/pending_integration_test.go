// Purpose: Verify deferred messages start only after the preceding HTTP generation completes.
// Pattern: HTTP and session integration test.
// Usage: go test ./internal/tui/session -run TestPendingRunsAfterCurrentAnswer.
// Related: internal/tui/session/pending.go, internal/tui/session/tasks.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
)

func TestPendingRunsAfterCurrentAnswer(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	a.ctx = ctx
	started, release := make(chan struct{}), make(chan struct{})
	var requests atomic.Int32
	var continued atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []modelapi.Message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if requests.Add(1) == 1 {
			close(started)
			select {
			case <-release:
			case <-ctx.Done():
				return
			}
		} else {
			answer, next := false, false
			for _, m := range request.Messages {
				answer = answer || m.Role == "assistant" && strings.Contains(m.Content, "first answer")
				next = next || m.Role == "user" && strings.Contains(m.Content, "next request")
			}
			continued.Store(answer && next)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"first answer"},"finish_reason":"stop"}]}`)
	}))
	t.Cleanup(func() {
		cancel()
		a.workers.Wait()
		if a.client != nil {
			a.client.Close()
		}
		server.Close()
	})
	connection, err := modelconn.New(modelconn.OpenAICompatible, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	a.client, err = httpclient.NewClient(connection, "", "")
	if err != nil {
		t.Fatal(err)
	}
	a.selection = modelconn.Selection{Connection: connection, Model: "mock"}
	a.startTask("first request")
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	a.editor.Set("next request")
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if requests.Load() != 1 || a.pending != "next request" {
		t.Fatal("request started before completion")
	}
	a.editor.Set("unsent draft")
	close(release)
	drainTask(t, a, false)
	if requests.Load() != 2 || !continued.Load() || a.pending != "" || a.editor.String() != "unsent draft" {
		t.Fatal("deferred input order or draft lost")
	}
}
