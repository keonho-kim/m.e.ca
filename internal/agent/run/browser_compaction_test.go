// Purpose: Exercise rejected browser pages through HTTP generation, SQLite compaction and forward-only continuation.
// Pattern: Scripted local provider with a deterministic page fixture or an opt-in real Chrome session.
// Usage: go test ./internal/agent/run -run TestBrowserCompactionFlow; MECA_TEST_NATIVE_COMPACTION=1 exercises Chrome.
// Related: internal/storage/sqlite/memory_boundary.go, internal/tools/browser/paging.go, internal/provider/httpclient/request.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local"
	"github.com/keonho-kim/meca/internal/tools/local/catalog"
)

type inspectTools struct{ *executor }

func (e inspectTools) Definitions() []toolapi.Definition {
	return catalog.Definitions("browser_inspect")
}

func TestBrowserCompactionFlow(t *testing.T) {
	ctx := t.Context()
	project, home := t.TempDir(), t.TempDir()
	sessions, err := sqlite.OpenSessions(filepath.Join(home, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	db, err := sessions.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id := db.Session().ID
	if err := db.AppendMessage(ctx, id, modelapi.Message{Role: "user", Content: "archive this earlier discussion"}); err != nil {
		t.Fatal(err)
	}

	// Simulate the wire exchange: inspect, continue, reject the second page, summarize, resume, finish.
	step := 0
	var pinned struct {
		ResultID string `json:"result_id"`
		Revision string `json:"revision"`
		Next     int    `json:"next_offset"`
		Text     string `json:"text"`
	}
	var pending string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		step++
		var req struct {
			Max      int `json:"max_tokens"`
			Messages []struct{ Role, Content string }
			Tools    []json.RawMessage
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			w.WriteHeader(500)
			return
		}
		toolResult := ""
		for _, m := range req.Messages {
			if strings.Contains(m.Content, "NEVER_REQUESTED_SUFFIX") {
				t.Errorf("step %d received the whole source", step)
			}
			if m.Role == "tool" {
				toolResult = m.Content
			}
			if step >= 3 && strings.Contains(m.Content, "PAGE_ONE_BODY") {
				t.Errorf("step %d replayed consumed page one", step)
			}
		}
		answer := func(text string) {
			fmt.Fprintf(w, `{"choices":[{"message":{"content":%q},"finish_reason":"stop"}]}`, text)
		}
		call := func(args map[string]any) {
			b, _ := json.Marshal(args)
			fmt.Fprintf(w, `{"choices":[{"message":{"tool_calls":[{"id":"call_%d","type":"function","function":{"name":"browser_inspect","arguments":%q}}]},"finish_reason":"tool_calls"}]}`, step, string(b))
		}
		next := func() {
			call(map[string]any{"result_id": pinned.ResultID, "revision": pinned.Revision, "offset": pinned.Next, "limit": 256})
		}
		switch step {
		case 1:
			call(map[string]any{"view": "text", "selector": "#proof", "limit": 256})
		case 2:
			if err := json.Unmarshal([]byte(toolResult), &pinned); err != nil || len(pinned.Text) != 256 || !strings.Contains(pinned.Text, "PAGE_ONE_BODY") || pinned.Next != 256 {
				t.Errorf("invalid first page: %v", err)
			}
			next()
		case 3:
			pending = toolResult
			if !strings.Contains(pending, "PAGE_TWO_BODY") {
				t.Error("missing current page")
			}
			w.WriteHeader(400)
			fmt.Fprint(w, `{"error":{"type":"BadRequestError","code":400,"message":"You requested 196608 output tokens and your prompt contains at least 65537 input tokens for a total of at least 262145 tokens."}}`)
		case 4:
			if len(req.Tools) != 0 || !strings.Contains(req.Messages[0].Content, "Summarize") {
				t.Error("provider rejection did not trigger a summary first")
			}
			for _, m := range req.Messages {
				if strings.Contains(m.Content, "PAGE_TWO_BODY") {
					t.Error("unread page archived")
				}
			}
			answer("Continue the browser inspection.")
		case 5:
			if req.Max != 196608 || toolResult != pending {
				t.Error("lost or changed the pending page or output policy")
			}
			if err := json.Unmarshal([]byte(toolResult), &pinned); err != nil || pinned.Next != 512 {
				t.Error("lost continuation cursor")
			}
			next()
		case 6:
			if !strings.Contains(toolResult, "PAGE_THREE_BODY") {
				t.Error("did not continue at page three")
			}
			answer("done")
		default:
			t.Error("unexpected generation")
			answer("stop")
		}
	}))
	defer provider.Close()
	conn, err := connection.New(connection.LiteLLM, provider.URL)
	if err != nil {
		t.Fatal(err)
	}
	model, err := httpclient.NewClient(conn, "")
	if err != nil {
		t.Fatal(err)
	}
	defer model.Close()
	redact := func(s string) string { return s }
	runner := Runner{Generator: model, Journal: db, Session: id, Model: "fixture", Provider: conn.ID, ProjectRoot: project, Redact: redact, Settings: policy.Settings{Scope: policy.System},
		Memory: &compact.Controller{Repository: db, Session: id, Model: "fixture", Prompt: prompt.Defaults().Compaction, Settings: compact.Defaults(), Redact: redact}}
	var body strings.Builder
	pages := []string{"PAGE_ONE_BODY", "PAGE_TWO_BODY", "PAGE_THREE_BODY"}
	ex := &executor{}
	for i, marker := range pages {
		text := marker + strings.Repeat("x", 256-len(marker))
		body.WriteString(text)
		result, _ := json.Marshal(map[string]any{"text": text, "result_id": "result_test", "revision": "revision", "offset": i * 256, "next_offset": (i + 1) * 256})
		ex.outcomes = append(ex.outcomes, toolapi.Result{Content: result})
	}
	runner.Tools = inspectTools{ex}
	if os.Getenv("MECA_TEST_NATIVE_COMPACTION") == "1" {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<pre id='proof'>%s%sNEVER_REQUESTED_SUFFIX</pre>", body.String(), strings.Repeat("x", 128<<10))
		}))
		defer server.Close()
		browser, err := local.NewExecutor(project, home, filepath.Join(home, ".meca"), redact, local.Options{})
		if err != nil {
			t.Fatal(err)
		}
		defer browser.Close()
		args, _ := json.Marshal(map[string]string{"action": "open", "url": server.URL})
		op, err := browser.Prepare(ctx, runner.Settings, toolapi.Call{ID: "open", Name: "browser_action", Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := op.Execute(ctx, true); err != nil {
			t.Fatal(err)
		}
		runner.Tools = browser
	}
	summaries, inspections := 0, 0
	result, err := runner.Run(ctx, "Read three pages", func(context.Context, toolapi.Approval) (bool, error) { return true, nil }, func(ev runrecord.Event) error {
		switch ev.Kind {
		case "retry":
			t.Error("budget retry event")
		case "compaction":
			summaries++
		case "tool_succeeded":
			inspections++
		}
		return nil
	})
	if err != nil || result.Text != "done" || step != 6 || summaries != 1 || inspections != 3 {
		t.Fatalf("steps=%d summaries=%d inspections=%d err=%v", step, summaries, inspections, err)
	}
}
