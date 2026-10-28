// Purpose: Verify queued requests execute sequentially with complete conversation history and released references.
// Pattern: HTTP and session integration test.
// Usage: go test ./internal/tui/session -run TestPendingRunsAfterCurrentAnswer.
// Related: internal/tui/session/pending.go, internal/tui/session/tasks.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/meca/internal/tui/composer"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
)

func TestPendingRunsAfterCurrentAnswer(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	a.ctx = ctx
	started, release := make(chan struct{}), make(chan struct{})
	sequence := []string{"first request", "next request", ":q", "fourth request"}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []modelapi.Message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		attempt := int(requests.Add(1))
		if attempt > len(sequence) {
			t.Error("unexpected generation beyond the queued requests")
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		var received []string
		for _, m := range request.Messages {
			if m.Role == "user" {
				for _, text := range sequence {
					if strings.Contains(m.Content, text) {
						received = append(received, text)
						break
					}
				}
			}
		}
		if !slices.Equal(received, sequence[:attempt]) {
			t.Errorf("request %d history=%v; want %v", attempt, received, sequence[:attempt])
		}
		for previous := 1; previous < attempt; previous++ {
			found := false
			for _, m := range request.Messages {
				found = found || m.Role == "assistant" && m.Content == fmt.Sprintf("answer %d", previous)
			}
			if !found {
				t.Errorf("request %d lost answer %d", attempt, previous)
			}
		}
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
	a.startRequest(composer.Capture(sequence[0], a.selectedPaths()))
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	for _, text := range sequence[1:] {
		if text == ":q" {
			a.page, a.browser.Input = "files", true
		}
		a.editor.Set(text)
		a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
		if a.fileSendConfirm {
			a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
		}
	}
	if requests.Load() != 1 || !slices.Equal(queuedTexts(a.queued.Messages), sequence[1:]) {
		t.Fatal("request started before completion")
	}
	retained := a.queued.Messages
	a.editor.Set("unsent draft")
	close(release)
	drainTask(t, a, false)
	if requests.Load() != int32(len(sequence)) || a.queued.Messages != nil || a.editor.String() != "unsent draft" {
		t.Fatal("deferred input order or draft lost")
	}
	for _, text := range retained {
		if text.Text != "" || text.Paths != nil {
			t.Fatal("completed queue retained request references")
		}
	}
	history, err := a.opts.Store.Messages(a.ctx, a.opts.Session.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	var recorded []string
	for _, message := range history {
		if message.Role == "user" {
			recorded = append(recorded, message.Content)
		}
	}
	if !slices.Equal(recorded, sequence) {
		t.Fatalf("stored request order=%v; want %v", recorded, sequence)
	}
}
