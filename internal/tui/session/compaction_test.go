// Purpose: Verify compaction context and account its model calls separately from the compaction threshold.
// Pattern: HTTP/SQLite task integration test.
// Usage: go test ./internal/tui/session -run TestCompactionContext.
// Related: internal/tui/session/tasks.go, internal/agent/compact/controller.go, internal/storage/sqlite/memory.go.
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

	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
)

func TestCompactionContext(t *testing.T) {
	a := fixture(t)
	a.opts.Config.Compaction.KeepRecentTokens = 0
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	a.ctx = ctx
	t.Cleanup(func() {
		cancel()
		a.workers.Wait()
		if a.connection.client != nil {
			a.connection.client.Close()
		}
	})
	if err := a.opts.Store.Commit(ctx, a.opts.Session.ID, []runrecord.Entry{{Message: &modelapi.Message{Role: "user", Content: "Earlier goal: update src/auth.go"}}}); err != nil {
		t.Fatal(err)
	}
	if err := a.opts.Store.AddContextTokens(ctx, a.opts.Session.ID, compact.TokenCount{Amount: 196608, Kind: "request"}); err != nil {
		t.Fatal(err)
	}
	const summaryMemo = "- Request: Update src/auth.go.\n- Result: No code changes yet.\n- Next: Continue the requested implementation."
	var calls atomic.Int64
	var decisions atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body struct {
			Messages []modelapi.Message `json:"messages"`
			Effort   string             `json:"reasoning_effort"`
			Tools    []json.RawMessage  `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body.Effort != "medium" {
			t.Errorf("effort lost in generation/compaction: %q", body.Effort)
		}
		reply := "Remembered the earlier goal"
		if len(body.Tools) == 0 {
			if !strings.Contains(body.Messages[1].Content, `"ActiveRequest":"Continue the earlier goal"`) {
				t.Error("active request missing from summary input")
			}
			reply = summaryMemo
		} else {
			if !strings.Contains(body.Messages[1].Content, summaryMemo) {
				t.Error("committed memory missing")
			}
			want := "Continue the earlier goal"
			if decisions.Add(1) > 1 {
				want = "New goal after compaction"
			}
			for _, message := range body.Messages {
				if message.Role == "user" && message.Content == "Earlier goal: update src/auth.go" {
					t.Error("summarized source replayed as raw history")
				}

			}
			if last := body.Messages[len(body.Messages)-1]; last.Role != "user" || !strings.HasSuffix(last.Content, want) {
				t.Error("current question lost inside compaction")
			}
		}
		w.Header().Set("Content-Type", "application/json")
		encoded, err := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": reply}, "finish_reason": "stop"}}, "usage": map[string]int{"prompt_tokens": 20, "completion_tokens": 7}})
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Fprint(w, string(encoded))
	}))
	t.Cleanup(server.Close)
	connection, err := modelconn.New(modelconn.VLLM, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	a.connection.client, err = httpclient.NewClient(connection, "")
	if err != nil {
		t.Fatal(err)
	}
	on := true
	effort := "medium"
	a.connection.selection = modelconn.Selection{Connection: connection, Model: "Qwen/Qwen3.8-27B", Thinking: &on, ReasoningEffort: &effort}
	a.startRequest(requestinput.Capture("Continue the earlier goal", a.selectedPaths()))
	drainTask(t, a, false)
	memory, err := a.opts.Store.Memory(ctx, a.opts.Session.ID)
	if err != nil || memory.Revision != 1 || !strings.Contains(memory.Markdown, "src/auth.go") {
		t.Fatalf("memory: %+v %v", memory, err)
	}
	a.startRequest(requestinput.Capture("New goal after compaction", a.selectedPaths()))
	drainTask(t, a, false)
	usage, err := a.opts.Store.MemoryWatermark(ctx, a.opts.Session.ID, memory.UsageThrough)
	if err != nil || usage.Tokens != (20+7) {
		t.Fatalf("request input and output not counted once: %+v %v", usage, err)
	}
	total, err := a.opts.Store.TokenUsage(ctx, a.opts.Session.ID)
	if err != nil || calls.Load() <= 2 || total.Input != 20*calls.Load() || total.Output != 7*calls.Load() || a.chat.displayTokens() != total {
		t.Fatalf("compaction and conversation usage=%+v display=%+v err=%v", total, a.chat.displayTokens(), err)
	}
	a.chat.tokenTotal = modelapi.TokenUsage{}
	if err := a.restore(); err != nil {
		t.Fatal(err)
	}
	if a.chat.displayTokens() != total {
		t.Fatal("usage not restored with conversation")
	}
}

func TestManualCompactionCommand(t *testing.T) {
	a := fixture(t)
	ctx := t.Context()
	a.ctx = ctx
	t.Cleanup(a.workers.Wait)
	if err := a.opts.Store.Commit(ctx, a.opts.Session.ID, []runrecord.Entry{
		{Message: &modelapi.Message{Role: "user", Content: "Keep the blue file."}},
	}); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokenize" {
			fmt.Fprint(w, `{"count":1200,"max_model_len":32768}`)
			return
		}
		calls.Add(1)
		var body struct {
			Messages  []modelapi.Message
			MaxTokens int `json:"max_tokens"`
			Tools     []json.RawMessage
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body.MaxTokens != 2293 {
			t.Errorf("summary requested %d output tokens, want 2293 for the serving context", body.MaxTokens)
		}
		if len(body.Messages) != 2 || len(body.Tools) != 0 ||
			!strings.Contains(body.Messages[1].Content, "Keep the blue file.") {
			t.Error("manual command did not send a summary-only request")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"Preserve the blue file."},"finish_reason":"stop"}]}`)
	}))
	t.Cleanup(server.Close)
	connection, err := modelconn.New(modelconn.VLLM, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	a.connection.client, err = httpclient.NewClient(connection, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.connection.client.Close)
	model := modelconn.ModelInfo{ID: "fixture", ContextWindow: 65536}
	if err := a.connection.client.BindModel(model); err != nil {
		t.Fatal(err)
	}
	a.connection.selection = modelconn.Selection{Connection: connection, Model: model.ID, Details: &model}
	a.compose.editor.Set("!compact")
	a.submit()
	drainTask(t, a, false)
	memory, err := a.opts.Store.Memory(ctx, a.opts.Session.ID)
	if err != nil || memory.Revision != 1 || memory.Markdown != "Preserve the blue file." {
		t.Fatalf("memory = %+v, error %v", memory, err)
	}
	if calls.Load() != 1 || a.compose.editor.String() != "" || a.status != a.language.Text("Conversation context compacted.") {
		t.Fatalf("command status=%q, draft=%q, provider calls=%d", a.status, a.compose.editor.String(), calls.Load())
	}
}
