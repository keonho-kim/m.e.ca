// Purpose: Verify compaction is triggered by accumulated tokens and fed into the next actual model request.
// Pattern: HTTP/SQLite task integration test.
// Usage: go test ./internal/tui/session -run TestCompactionContext.
// Related: internal/tui/session/tasks.go, internal/agent/compact/controller.go, internal/storage/sqlite/memory.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCompactionContext(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	a.ctx = ctx
	t.Cleanup(func() {
		cancel()
		a.workers.Wait()
		if a.client != nil {
			a.client.Close()
		}
	})
	if err := a.opts.Store.AppendMessage(ctx, a.opts.Session.ID, agent.Message{Role: "user", Content: "Earlier goal: update src/auth.go"}); err != nil {
		t.Fatal(err)
	}
	if err := a.opts.Store.AddContextTokens(ctx, a.opts.Session.ID, compact.TokenCount{Amount: 256000, Kind: "model"}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []agent.Message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		reply := "DONE\nRemembered the earlier goal"
		if strings.Contains(body.Messages[0].Content, "conversation compaction prompt") {
			var b strings.Builder
			for _, name := range []string{"Goals", "Task history", "Steps and changed files", "Verification", "Decisions and constraints", "Pending work and blockers", "Continuation"} {
				b.WriteString("## " + name + "\n- Earlier goal: update src/auth.go; pending until verified.\n")
			}
			reply = b.String()
		} else if !strings.Contains(body.Messages[1].Content, "Earlier goal: update src/auth.go") {
			t.Error("committed compaction memory missing from next model context")
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
	connection, err := modelconn.New(modelconn.OpenAICompatible, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	a.client, err = httpclient.NewClient(connection, "")
	if err != nil {
		t.Fatal(err)
	}
	a.selection = sqlite.Selection{Connection: connection, Model: "mock"}
	a.startTask("Continue the earlier goal", nil)
	drainTask(t, a, false)
	memory, err := a.opts.Store.Memory(ctx, a.opts.Session.ID)
	if err != nil || memory.Revision != 1 || !strings.Contains(memory.Markdown, "src/auth.go") {
		t.Fatalf("memory: %+v %v", memory, err)
	}
	usage, err := a.opts.Store.MemoryWatermark(ctx, a.opts.Session.ID, memory.UsageThrough)
	if err != nil || usage.Tokens != 7 {
		t.Fatalf("new model output not counted once: %+v %v", usage, err)
	}
}
