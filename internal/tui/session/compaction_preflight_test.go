// Purpose: Verify automatic compaction uses the next request's tokenizer count before generation.
// Pattern: Loopback provider and SQLite integration test.
// Usage: go test ./internal/tui/session -run TestCompactionUsesPreparedInput.
// Related: internal/agent/run/loop.go, internal/provider/httpclient/context.go, internal/tui/session/compaction_test.go.
package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
)

func TestCompactionUsesPreparedInput(t *testing.T) {
	const previous = "Earlier request with a result to retain"
	const objective = "Continue the current request"
	const summary = "Retain the earlier result."
	compacted := []string{"tokenize history", "tokenize summary", "send summary", "tokenize compacted", "send compacted"}
	uncompacted := []string{"tokenize history", "send history"}
	for _, tc := range []struct {
		name       string
		window     int
		input      int
		prior      int64
		retainTail bool
		maxBytes   int
		want       []string
	}{
		{"at threshold", 65536, 49152, 30000, false, 0, compacted},
		{"below threshold with high previous usage", 65536, 49151, 125000, false, 0, uncompacted},
		{"75 percent of 256K window", 262144, 196608, 30000, false, 0, compacted},
		{"odd window below 75 percent", 65537, 49152, 30000, false, 0, uncompacted},
		{"only retained history", 65536, 65280, 30000, true, 0, uncompacted},
		{"input fills window", 65536, 65536, 30000, false, 0, compacted},
		{"history exceeds local byte limit", 65536, 0, 30000, false, 64 * 1024, compacted[1:]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := fixture(t)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			a.ctx = ctx
			if !tc.retainTail {
				a.opts.Config.Compaction.KeepRecentTokens = 0
			}
			old := previous
			if tc.maxBytes > 0 {
				a.opts.Config.Loop.MaxContextBytes = tc.maxBytes
				old += strings.Repeat(" source", tc.maxBytes/7+1)
			}
			store, session := a.opts.Store, a.opts.Session.ID
			if err := store.Commit(ctx, session, []runrecord.Entry{
				{Message: &modelapi.Message{Role: "user", Content: old}},
				{Message: &modelapi.Message{Role: "assistant", Content: "Earlier result confirmed."}},
			}); err != nil {
				t.Fatal(err)
			}
			if err := store.AddContextTokens(ctx, session, compact.TokenCount{Amount: tc.prior, Kind: "request"}); err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			var trace []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct{ Messages []modelapi.Message }
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Messages) < 2 {
					t.Errorf("invalid request: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				name, count, reply := "history", tc.input, "Answer"
				var summaryInput struct {
					NextSource json.RawMessage
				}
				isSummary := json.Unmarshal([]byte(body.Messages[1].Content), &summaryInput) == nil && summaryInput.NextSource != nil
				if isSummary {
					name, count, reply = "summary", 1500, summary
					if !strings.Contains(body.Messages[1].Content, old) {
						t.Error("summary lost its source")
					}
				} else {
					var hasOld, hasSummary, hasObjective bool
					for _, message := range body.Messages {
						hasOld = hasOld || message.Content == old
						hasSummary = hasSummary || strings.Contains(message.Content, summary)
						hasObjective = hasObjective || message.Content == objective
					}
					if hasSummary {
						name, count = "compacted", 30000
					}
					if !hasObjective || hasOld == hasSummary {
						t.Error("agent request lost source, summary or current objective")
					}
				}
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/tokenize" {
					trace = append(trace, "tokenize "+name)
					fmt.Fprintf(w, `{"count":%d,"max_model_len":%d}`, count, tc.window)
					return
				}
				trace = append(trace, "send "+name)
				fmt.Fprintf(w, `{"choices":[{"message":{"content":%q},"finish_reason":"stop"}]}`, reply)
			}))
			t.Cleanup(server.Close)
			connection, err := modelconn.New(modelconn.VLLM, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			client, err := httpclient.NewClient(connection, "")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(client.Close)
			t.Cleanup(a.workers.Wait)
			info := modelconn.ModelInfo{ID: "fixture", ContextWindow: max(131072, tc.window)}
			if err := client.BindModel(info); err != nil {
				t.Fatal(err)
			}
			a.connection.client = client
			a.connection.selection = modelconn.Selection{Connection: connection, Model: info.ID, Details: &info}
			a.startRequest(requestinput.Capture(objective, nil))
			drainTask(t, a, false)
			mu.Lock()
			if !slices.Equal(trace, tc.want) {
				t.Errorf("provider calls=%v, want %v", trace, tc.want)
			}
			mu.Unlock()
			var exported bytes.Buffer
			if err := store.(*sqlite.Store).Export(ctx, session, &exported); err != nil {
				t.Fatal(err)
			}
			decoder := json.NewDecoder(&exported)
			var agent *runrecord.Generation
			for {
				var entry struct{ Body runrecord.Event }
				if err := decoder.Decode(&entry); err == io.EOF {
					break
				} else if err != nil {
					t.Fatal(err)
				}
				if g := entry.Body.Generation; g != nil && g.Role == "agent" && entry.Body.Kind == "generation_finished" {
					agent = g
				}
			}
			if agent == nil || agent.Context == nil || agent.Context.Source != "tokenizer" || agent.Context.Window != tc.window || agent.Context.OutputBudget <= 0 {
				t.Fatalf("missing current request budget: %+v", agent)
			}
			if slices.Contains(tc.want, "send summary") {
				before := agent.BeforeCompaction
				if before == nil || before.Window != tc.window && tc.maxBytes == 0 || agent.Context.Input != 30000 {
					t.Fatalf("compaction accounting: before=%+v after=%+v", before, agent.Context)
				}
				if tc.input > 0 && (before.Source != "tokenizer" || before.Input != int64(tc.input)) {
					t.Fatalf("compaction trigger=%+v want %d", before, tc.input)
				}
			} else if agent.BeforeCompaction != nil || agent.Context.Input != int64(tc.input) {
				t.Fatalf("unexpected compaction accounting: %+v", agent)
			}
			checkpoint, err := store.LatestRun(ctx, session)
			wantCalls := 1
			if slices.Contains(tc.want, "send summary") {
				wantCalls++
			}
			if err != nil || checkpoint == nil || checkpoint.ModelCalls != wantCalls {
				t.Fatalf("prepared requests counted as generations: checkpoint=%+v, want %d model calls, err=%v", checkpoint, wantCalls, err)
			}
		})
	}
}
