// Purpose: Verify saved conversation remains in actual model requests and newest questions are distinct.
// Pattern: HTTP/SQLite integration tests.
// Usage: go test ./internal/tui/session -run TestConversationContext.
// Related: internal/tui/session/tasks.go, internal/agent/run/context.go, internal/storage/sqlite/messages.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestConversationContextIncludesAllBeforeCompaction(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	a.ctx = ctx
	t.Cleanup(func() {
		cancel()
		a.workers.Wait()
		if a.client != nil {
			a.client.Close()
		}
	})
	long := strings.Repeat("long history ", 500) + "old detail preserved"
	for i := 0; i < 1002; i++ {
		message := modelapi.Message{Role: "user", Content: fmt.Sprintf("earlier-%d", i)}
		if i == 0 {
			message.Content = long
		}
		if err := a.opts.Store.AppendMessage(ctx, a.opts.Session.ID, message); err != nil {
			t.Fatal(err)
		}
	}
	call := toolapi.Call{ID: "earlier-call", Name: "read_file", Arguments: json.RawMessage(`{"path":"README.md"}`)}
	for _, message := range []modelapi.Message{{Role: "assistant", Calls: []toolapi.Call{call}}, {Role: "tool", ToolCallID: call.ID, ToolName: call.Name, Content: `{"content":"exact runtime output"}`}} {
		if err := a.opts.Store.AppendMessage(ctx, a.opts.Session.ID, message); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.restore(); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []modelapi.Message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if respondGoalAnalysis(t, w, request.Messages) {
			return
		}
		call := calls.Add(1)
		newest := "new question"
		if call == 2 {
			newest = "different question"
		}
		oldFound, newFound, previousAnswer, toolOutput := false, false, false, false
		for _, message := range request.Messages {
			if len(message.Calls) > 0 || message.Role == "tool" {
				t.Error("native tool protocol leaked into text dialogue")
			}
			if strings.Contains(message.Content, "exact runtime output") && strings.Contains(message.Content, "earlier-call") {
				toolOutput = true
			}
			if message.Role == "user" && message.Content == long {
				oldFound = true
			}
			if message.Role == "user" && message.Content == newest {
				newFound = true
			}
			if message.Role == "assistant" && message.Content == "new answer" {
				previousAnswer = true
			}
		}
		if !oldFound || !newFound || !toolOutput || len(request.Messages) < 1006 {
			t.Errorf("conversation omitted: old=%t new=%t messages=%d", oldFound, newFound, len(request.Messages))
		}
		if call == 2 && !previousAnswer {
			t.Error("previous response missing from next request")
		}
		reply := "DONE\nnew answer"
		if call == 2 {
			reply = "DONE\ndifferent answer"
		}
		w.Header().Set("Content-Type", "application/json")
		encoded, err := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": reply}, "finish_reason": "stop"}}})
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Fprint(w, string(encoded))
	}))
	t.Cleanup(server.Close)
	connection, err := modelconn.New(modelconn.OpenAICompatible, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	a.client, err = httpclient.NewClient(connection, "", "")
	if err != nil {
		t.Fatal(err)
	}
	a.selection = modelconn.Selection{Connection: connection, Model: "context-model"}
	for _, question := range []string{"new question", "different question"} {
		a.stream, a.thinking = "stale answer", "stale thinking"
		a.startTask(question, nil)
		if a.stream != "" || a.thinking != "" {
			t.Fatal("previous response retained on new task")
		}
		drainTask(t, a, false)
	}
	if calls.Load() != 2 {
		t.Fatalf("requests=%d", calls.Load())
	}
}
