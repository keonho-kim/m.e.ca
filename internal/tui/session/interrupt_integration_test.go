// Purpose: Verify cancelled streamed answers survive in the next request's persisted context.
// Pattern: HTTP and session integration test.
// Usage: go test ./internal/tui/session -run TestEscapeRetainsStreamForNextRequest.
// Related: internal/tui/session/tasks.go, internal/tui/session/activity.go.
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

func TestEscapeRetainsStreamForNextRequest(t *testing.T) {
	for _, answer := range []bool{false, true} {
		t.Run(fmt.Sprintf("answer=%t", answer), func(t *testing.T) { checkInterruptedContext(t, answer) })
	}
}

func checkInterruptedContext(t *testing.T, answer bool) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	a.ctx = ctx
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
			w.Header().Set("Content-Type", "text/event-stream")
			content := ""
			if answer {
				content = "partial answer"
			}
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"reasoning\":\"partial thinking\",\"content\":%q}}]}\n\n", content)
			w.(http.Flusher).Flush()
			select {
			case <-r.Context().Done():
			case <-ctx.Done():
			}
			return
		}
		first, partial, next, thinking := false, false, false, false
		for _, m := range request.Messages {
			first = first || m.Role == "user" && strings.Contains(m.Content, "first request")
			partial = partial || m.Role == "assistant" && strings.Contains(m.Content, "partial answer")
			thinking = thinking || strings.Contains(m.Content, "partial thinking")
			next = next || m.Role == "user" && strings.Contains(m.Content, "continue now")
		}
		continued.Store(first && (partial || !answer) && next && thinking)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"continued answer"},"finish_reason":"stop"}]}`)
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
	a.startTask("first request", nil)
	for !strings.Contains(a.thinking, "partial thinking") {
		select {
		case n := <-a.events:
			a.receive(n)
			if n.Done {
				t.Fatalf("ended before stream: %v", n.Err)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	for a.busy {
		select {
		case n := <-a.events:
			a.receive(n)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	if a.exit || a.ctx.Err() != nil {
		t.Fatal("Esc closed session")
	}
	a.startTask("continue now", nil)
	drainTask(t, a, false)
	if !continued.Load() || requests.Load() != 2 {
		t.Fatalf("context lost or request replayed: %d requests", requests.Load())
	}
}
