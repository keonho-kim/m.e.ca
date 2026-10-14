// Purpose: Verify Plan-mode greetings remain ordinary conversation and only a completed proposal opens approval.
// Pattern: Provider-to-terminal integration test.
// Usage: go test ./internal/tui/session -run TestPlanConversation.
// Related: internal/agent/run/planning.go, internal/tui/session/runtime_events.go, internal/tui/session/worker_result.go.
package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
)

func TestPlanConversationGreetsBeforeProposing(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	a.ctx = ctx
	t.Cleanup(func() { cancel(); a.shutdown() })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct{ Messages []modelapi.Message }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		reply := "DONE\nHello! 어떤 작업을 계획할까요?"
		for _, message := range request.Messages {
			if message.Role == "user" && message.Content == "plan now" {
				reply = "DONE\n{\"objective\":\"inspect project\",\"steps\":[\"Read the entry point\",\"Propose changes\"]}"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": reply}, "finish_reason": "stop"}}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	conn, err := connection.New(connection.OpenAICompatible, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	a.client, err = httpclient.NewClient(conn, "", "")
	if err != nil {
		t.Fatal(err)
	}
	a.selection = connection.Selection{Connection: conn, Model: "test"}
	enterPlanCommand(a)
	initial := *a.plan
	a.startTask("hello", nil)
	drainTask(t, a, false)
	if !a.planMode || a.plan.Status != planning.Planning || a.plan.Revision != initial.Revision || a.page != "chat" {
		t.Fatal("greeting changed or completed Plan mode")
	}
	shown := false
	for _, line := range a.output.Lines(100) {
		if strings.Contains(line.Text, "Hello!") {
			shown = true
		}
	}
	if !shown {
		t.Fatal("greeting was hidden")
	}
	a.startTask("plan now", nil)
	drainTask(t, a, false)
	if a.plan.Status != planning.AwaitingApproval || a.page != "planapproval" || a.plan.Revision != initial.Revision+1 {
		t.Fatalf("completed Plan not offered: %+v %s", a.plan, a.page)
	}
}
