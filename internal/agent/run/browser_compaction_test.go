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

type inspectTools struct {
	*executor
	otherTool bool
}

func (e inspectTools) Definitions() []toolapi.Definition {
	return catalog.Definitions(toolapi.BrowserInspect, "list_skills")
}

func (e inspectTools) BrowserCursors(context.Context) ([]toolapi.BrowserCursor, error) {
	page := e.executed
	if e.otherTool && page >= 3 {
		page--
	}
	if page == 0 || page > 2 {
		return nil, nil
	}
	return []toolapi.BrowserCursor{{ResultID: "result_test", Revision: "revision", NextOffset: page * 256}}, nil
}

func TestBrowserCompactionFlow(t *testing.T) {
	t.Run("unread", func(t *testing.T) { testBrowserCompaction(t, false) })
	t.Run("after unrelated tool", func(t *testing.T) { testBrowserCompaction(t, true) })
}

func testBrowserCompaction(t *testing.T, otherTool bool) {
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
		phase := step
		if otherTool && step > 3 {
			phase--
		}
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
		var live []toolapi.BrowserCursor
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
			if strings.HasPrefix(m.Content, "Unread browser observations.") {
				_, data, _ := strings.Cut(m.Content, "\n")
				if err := json.Unmarshal([]byte(data), &live); err != nil {
					t.Error(err)
				}
			}
		}
		answer := func(text string) {
			fmt.Fprintf(w, `{"choices":[{"message":{"content":%q},"finish_reason":"stop"}]}`, text)
		}
		call := func(name string, args map[string]any) {
			b, _ := json.Marshal(args)
			fmt.Fprintf(w, `{"choices":[{"message":{"tool_calls":[{"id":"call_%d","type":"function","function":{"name":%q,"arguments":%q}}]},"finish_reason":"tool_calls"}]}`, step, name, string(b))
		}
		next := func() {
			call(toolapi.BrowserInspect, map[string]any{"result_id": pinned.ResultID, "revision": pinned.Revision, "offset": pinned.Next, "limit": 256})
		}
		if otherTool && step == 3 {
			if !strings.Contains(toolResult, "PAGE_TWO_BODY") {
				t.Error("missing observation before unrelated tool")
			}
			call("list_skills", map[string]any{})
			return
		}
		switch phase {
		case 1:
			call(toolapi.BrowserInspect, map[string]any{"view": "text", "selector": "#proof", "limit": 256})
		case 2:
			if err := json.Unmarshal([]byte(toolResult), &pinned); err != nil || len(pinned.Text) != 256 || !strings.Contains(pinned.Text, "PAGE_ONE_BODY") || pinned.Next != 256 {
				t.Errorf("invalid first page: %v", err)
			}
			next()
		case 3:
			pending = toolResult
			if !otherTool && !strings.Contains(pending, "PAGE_TWO_BODY") {
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
			if len(live) != 1 || live[0].NextOffset != 512 {
				t.Errorf("lost live continuation cursor: %+v", live)
			} else {
				pinned.ResultID, pinned.Revision, pinned.Next = live[0].ResultID, live[0].Revision, live[0].NextOffset
			}
			if otherTool {
				for _, m := range req.Messages {
					if m.Role == "tool" && strings.Contains(m.Content, `"result_id"`) {
						t.Error("older browser result survived the compaction boundary")
					}
				}
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
	wantCalls, wantSteps := 3, 6
	if otherTool {
		runner.Memory.Settings.KeepRecentTokens = 1
		wantCalls, wantSteps = 4, 7
	}
	var body strings.Builder
	pages := []string{"PAGE_ONE_BODY", "PAGE_TWO_BODY", "PAGE_THREE_BODY"}
	ex := &executor{}
	for i, marker := range pages {
		if i == 2 && otherTool {
			ex.outcomes = append(ex.outcomes, toolapi.Result{Content: json.RawMessage(`{"Content":"A separate fact."}`)})
		}
		text := marker + strings.Repeat("x", 256-len(marker))
		body.WriteString(text)
		result, _ := json.Marshal(map[string]any{"text": text, "result_id": "result_test", "revision": "revision", "offset": i * 256, "next_offset": (i + 1) * 256})
		ex.outcomes = append(ex.outcomes, toolapi.Result{Content: result})
	}
	runner.Tools = inspectTools{executor: ex, otherTool: otherTool}
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
	if err != nil || result.Text != "done" || step != wantSteps || summaries != 1 || inspections != wantCalls {
		t.Fatalf("steps=%d summaries=%d inspections=%d err=%v", step, summaries, inspections, err)
	}
}
