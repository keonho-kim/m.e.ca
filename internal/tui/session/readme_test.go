// Purpose: Verify a file request receives its project root and reaches approved execution without AGENTS.md.
// Pattern: HTTP, filesystem and session integration test.
// Usage: go test ./internal/tui/session -run TestReadmeRequest.
// Related: internal/tui/session/tasks.go, internal/agent/run/task_context.go, internal/tools/local/instructions.go.
package session

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
	"time"

	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
)

func TestReadmeRequestWithoutProjectInstructions(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	a.ctx, a.settings.Scope = ctx, policy.System
	if _, err := os.Stat(filepath.Join(a.opts.Session.ProjectRoot, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no project instructions: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []modelapi.Message
			Tools    []struct{ Function struct{ Name string } }
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		question := request.Messages[len(request.Messages)-1].Content
		if strings.HasPrefix(question, "Work observations:") && !strings.Contains(question, fmt.Sprintf(`"project_root":%q`, a.opts.Session.ProjectRoot)) {
			t.Error("tool path base missing from model context")
		}
		usedTool := false
		for _, message := range request.Messages {
			usedTool = usedTool || message.Role == "tool"
		}
		reply, toolName := "Created README.md", ""
		switch {
		case !usedTool:
			toolName = "write_file"
		}
		message := map[string]any{"content": reply}
		if toolName != "" {
			arguments := map[string]any{"path": "README.md", "content": "# Project\n"}
			arguments["reason"] = "Perform the requested operation using the current observations."
			encoded, err := json.Marshal(arguments)
			if err != nil {
				t.Error(err)
				return
			}
			message = map[string]any{"tool_calls": []any{map[string]any{"id": fmt.Sprintf("call_%d", len(request.Messages)), "type": "function", "function": map[string]any{"name": toolName, "arguments": string(encoded)}}}}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": "stop"}}}); err != nil {
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
	a.startTask("Create README.md in the current directory")
	drainTask(t, a, true)
	content, err := os.ReadFile(filepath.Join(a.opts.Session.ProjectRoot, "README.md"))
	if err != nil || string(content) != "# Project\n" || a.status != "" {
		t.Fatalf("content=%q status=%q err=%v", content, a.status, err)
	}
	visible := false
	for _, block := range a.output.Blocks {
		if strings.Contains(block.Text, "+   1 | # Project") {
			visible = true
		}
	}
	if !visible {
		t.Fatal("created file body missing from transcript")
	}
	records, _, err := a.opts.Store.Timeline(ctx, a.opts.Session.ID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	persisted := false
	for _, record := range records {
		if record.Event.Kind == "tool_succeeded" && record.Event.Call != nil && record.Event.Call.Name == "write_file" {
			persisted = strings.Contains(record.Event.ChangePreview, "+# Project")
		}
	}
	if !persisted {
		t.Fatal("applied body not retained for resume")
	}

}
