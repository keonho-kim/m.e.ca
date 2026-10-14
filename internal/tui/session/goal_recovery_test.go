// Purpose: Verify a greeting survives malformed Goal analysis through HTTP and persistent session state.
// Pattern: HTTP-to-session integration test.
// Usage: go test ./internal/tui/session -run TestGoalAnalysisGreetingRecovery.
// Related: internal/tui/session/tasks.go, internal/agent/run/goal_model.go, internal/tui/session/tasks_test.go.
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
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
)

func TestGoalAnalysisGreetingRecovery(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	a.ctx = ctx
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct{ Messages []modelapi.Message }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		reply := "DONE\nHello! How can I help?"
		if strings.Contains(request.Messages[len(request.Messages)-1].Content, "\nGOAL ANALYSIS:") {
			reply = "Hello from an incorrectly formatted analysis"
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
	t.Cleanup(a.client.Close)
	t.Cleanup(func() { cancel(); a.workers.Wait() })
	a.selection = connection.Selection{Connection: conn, Model: "test-model"}
	a.startTask("Hello", nil)
	drainTask(t, a, false)
	var output strings.Builder
	for _, block := range a.output.Blocks {
		output.WriteString(block.Text)
	}
	if a.status != "" || strings.Count(output.String(), "Hello! How can I help?") != 1 || strings.Contains(output.String(), "incorrectly formatted analysis") {
		t.Fatalf("status=%q output=%q", a.status, output.String())
	}
}
