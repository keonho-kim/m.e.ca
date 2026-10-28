// Purpose: Verify interrupted output stays visible while resumed requests exclude unfinished thinking.
// Pattern: HTTP and session integration test.
// Usage: go test ./internal/tui/session -run TestEscapeRetainsStreamForNextRequest.
// Related: internal/tui/session/tasks.go, internal/tui/session/activity.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/meca/internal/tui/composer"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	"github.com/keonho-kim/meca/internal/tui/viewport"
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
		attempt := requests.Add(1)
		if attempt > 1 {
			for _, m := range request.Messages {
				if strings.Contains(m.Content+m.Reasoning, "partial thinking") || m.Role == "assistant" && strings.Contains(m.Content, "partial answer") {
					t.Error("unfinished thinking or a completed-looking draft replayed")
				}
			}
		}
		if attempt <= 3 {
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
			partial = partial || m.Role == "user" && strings.Contains(m.Content, "Unfinished assistant draft") && strings.Contains(m.Content, "partial answer")
			thinking = thinking || strings.Contains(m.Content+m.Reasoning, "partial thinking")
			next = next || m.Role == "user" && strings.Contains(m.Content, "continue now")
		}
		continued.Store(first && (partial || !answer) && next && !thinking)
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
	connection, err := modelconn.New(modelconn.OpenRouter, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	a.client, err = httpclient.NewClient(connection, "")
	if err != nil {
		t.Fatal(err)
	}
	a.selection = modelconn.Selection{Connection: connection, Model: "mock"}
	for _, input := range []string{"first request", "continue first", "continue second"} {
		a.startRequest(composer.Capture(input, a.selectedPaths()))
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
		for a.worker.active() {
			select {
			case n := <-a.events:
				a.receive(n)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		}
	}
	if a.exit || a.ctx.Err() != nil {
		t.Fatal("Esc closed session")
	}
	a.startRequest(composer.Capture("continue now", a.selectedPaths()))
	drainTask(t, a, false)
	if !continued.Load() || requests.Load() != 4 {
		t.Fatalf("context lost or request replayed: %d requests", requests.Load())
	}
	checkInterruptedArchive(t, a, answer)
}

func checkInterruptedArchive(t *testing.T, a *app, answer bool) {
	t.Helper()
	records, _, err := a.opts.Store.Timeline(a.ctx, a.opts.Session.ID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	thoughts, drafts := 0, 0
	for _, record := range records {
		if record.Kind == "event" && record.Event.Thinking == "partial thinking" {
			thoughts++
		}
		if record.Kind == "message" && record.Message.Content == "partial answer" {
			if !record.Message.Incomplete {
				t.Fatal("draft stored as completed")
			}
			drafts++
		}
	}
	wantDrafts := 0
	if answer {
		wantDrafts = 3
	}
	if thoughts != 3 || drafts != wantDrafts {
		t.Fatalf("archive lost: thoughts=%d drafts=%d", thoughts, drafts)
	}
	for _, restore := range []bool{false, true} {
		if restore {
			a.output = viewport.View{Follow: true}
			a.historyCursor = 0
			if err := a.loadHistory(); err != nil {
				t.Fatal(err)
			}
		}
		thoughts, drafts = 0, 0
		for _, block := range a.output.Blocks {
			if block.Thinking && strings.Contains(block.Text, "partial thinking") {
				thoughts++
			}
			if !block.Thinking && strings.Contains(block.Text, "partial answer") {
				drafts++
			}
		}
		if thoughts != 3 || drafts != wantDrafts {
			t.Fatalf("visible output lost (restore=%t): thoughts=%d drafts=%d", restore, thoughts, drafts)
		}
	}
}
