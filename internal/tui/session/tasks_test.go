// Purpose: Verify approved Plan execution and file approvals through real HTTP and SQLite adapters.
// Pattern: Integration behavior tests.
// Usage: go test ./internal/tui/session -run TestPlanExecution.
// Related: internal/tui/session/tasks.go, internal/provider/protocol/request.go, internal/storage/sqlite/events.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/policy"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlanExecutionWaitsForBothApprovals(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	a.ctx = ctx
	t.Cleanup(func() { cancel(); a.workers.Wait() })

	if err := os.WriteFile(filepath.Join(a.opts.Session.ProjectRoot, "AGENTS.md"), []byte("Project rule: use an actual verification command after changes."), 0600); err != nil {
		t.Fatal(err)
	}
	a.settings.Scope = policy.System
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []modelapi.Message `json:"messages"`
			Tools    json.RawMessage    `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			http.Error(w, "invalid", 400)
			return
		}
		if !strings.Contains(request.Messages[0].Content, "Project rule: use an actual verification command") {
			t.Error("project AGENTS.md missing from system prompt")
		}
		question := request.Messages[len(request.Messages)-1].Content
		reply := ""
		switch {
		case strings.HasPrefix(question, "REVIEW CANDIDATE "):
			reply = `{"decision":"pass"}`
		case strings.Contains(request.Messages[0].Content, "Plan mode:"):
			reply = `{"tool":"submit_plan"}`
		case strings.Contains(question, "execute_command result:"):
			reply = "Created and checked the file"
		case strings.Contains(question, "write_file result:"):
			reply = `{"tool":"execute_command"}`
		default:
			reply = `{"tool":"write_file"}`
		}
		message := map[string]any{"content": reply}
		if strings.HasPrefix(reply, `{"tool":`) {
			toolName := "write_file"
			arguments := map[string]any{"path": "output.txt", "content": "approved"}
			if reply == `{"tool":"execute_command"}` {
				toolName = "execute_command"
				arguments = map[string]any{"command": "test -f output.txt", "purpose": "check"}
			}
			if reply == `{"tool":"submit_plan"}` {
				toolName = "submit_plan"
				arguments = map[string]any{"objective": "create file", "steps": []string{"write output.txt and verify"}}
			}
			arguments["reason"] = "Perform the requested operation using the current observations."
			encoded, err := json.Marshal(arguments)
			if err != nil {
				t.Error(err)
				return
			}
			message = map[string]any{"tool_calls": []any{map[string]any{"id": fmt.Sprintf("call_%d", len(request.Messages)), "type": "function", "function": map[string]any{"name": toolName, "arguments": string(encoded)}}}}
		}
		w.Header().Set("Content-Type", "application/json")
		body, err := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": "stop"}}})
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Fprint(w, string(body))
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
	a.selection = modelconn.Selection{Connection: connection, Model: "mock"}
	a.editor.Set("!plan")
	a.submit()
	a.startTask("create file", nil)
	drainTask(t, a, false)
	path := filepath.Join(a.opts.Session.ProjectRoot, "output.txt")
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("planning wrote a file: %v", err)
	}
	if a.plan == nil || a.plan.Status != planning.AwaitingApproval || a.page != "planapproval" {
		t.Fatalf("plan state: %+v %s", a.plan, a.page)
	}
	a.approvePlan()
	drainTask(t, a, true)
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "approved" {
		t.Fatalf("approved execution: %q %v", content, err)
	}
	if a.plan.Status != planning.PlanSucceeded {
		t.Fatalf("Plan status: %s", a.plan.Status)
	}
	events, _, err := a.opts.Store.Timeline(ctx, a.opts.Session.ID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range events {
		if e.Event.Kind == "approved" {
			found = true
		}
	}
	if !found {
		t.Fatal("approval not persisted")
	}
}
func drainTask(t *testing.T, a *app, approve bool) {
	t.Helper()
	for a.busy {
		select {
		case n := <-a.events:
			a.receive(n)
			if n.Approval != nil {
				if !approve {
					t.Fatal("planning requested mutation approval")
				}
				a.decide(true)
			}
			if n.Done && n.Err != nil {
				t.Fatal(n.Err)
			}
		case <-a.ctx.Done():
			t.Fatal(a.ctx.Err())
		}
	}
}
