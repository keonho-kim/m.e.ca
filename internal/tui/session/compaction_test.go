// Purpose: Verify compaction context and account its model calls separately from the compaction threshold.
// Pattern: HTTP/SQLite task integration test.
// Usage: go test ./internal/tui/session -run TestCompactionContext.
// Related: internal/tui/session/tasks.go, internal/agent/compact/controller.go, internal/storage/sqlite/memory.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCompactionContext(t *testing.T) {
	a := fixture(t)
	a.opts.Config.Compaction.KeepRecentTokens = 0
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	a.ctx = ctx
	t.Cleanup(func() {
		cancel()
		a.workers.Wait()
		if a.client != nil {
			a.client.Close()
		}
	})
	if err := a.opts.Store.AppendMessage(ctx, a.opts.Session.ID, modelapi.Message{Role: "user", Content: "Earlier goal: update src/auth.go"}); err != nil {
		t.Fatal(err)
	}
	if err := a.opts.Store.AddContextTokens(ctx, a.opts.Session.ID, compact.TokenCount{Amount: 245760, Kind: "request"}); err != nil {
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
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body.Effort != "medium" {
			t.Errorf("effort lost in generation/compaction: %q", body.Effort)
		}
		reply := "Remembered the earlier goal"
		if strings.Contains(body.Messages[0].Content, "# Summarize work for continuation") {
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
	a.client, err = httpclient.NewClient(connection, "")
	if err != nil {
		t.Fatal(err)
	}
	on := true
	effort := "medium"
	a.selection = modelconn.Selection{Connection: connection, Model: "Qwen/Qwen3.8-27B", Thinking: &on, ReasoningEffort: &effort}
	a.startTask("Continue the earlier goal")
	drainTask(t, a, false)
	memory, err := a.opts.Store.Memory(ctx, a.opts.Session.ID)
	if err != nil || memory.Revision != 1 || !strings.Contains(memory.Markdown, "src/auth.go") {
		t.Fatalf("memory: %+v %v", memory, err)
	}
	a.startTask("New goal after compaction")
	drainTask(t, a, false)
	usage, err := a.opts.Store.MemoryWatermark(ctx, a.opts.Session.ID, memory.UsageThrough)
	if err != nil || usage.Tokens != (20+7) {
		t.Fatalf("request input and output not counted once: %+v %v", usage, err)
	}
	total, err := a.opts.Store.TokenUsage(ctx, a.opts.Session.ID)
	if err != nil || calls.Load() <= 2 || total.Input != 20*calls.Load() || total.Output != 7*calls.Load() || a.displayTokens() != total {
		t.Fatalf("compaction and conversation usage=%+v display=%+v err=%v", total, a.displayTokens(), err)
	}
	a.tokenTotal = modelapi.TokenUsage{}
	if err := a.restore(); err != nil {
		t.Fatal(err)
	}
	if a.displayTokens() != total {
		t.Fatal("usage not restored with conversation")
	}
}
