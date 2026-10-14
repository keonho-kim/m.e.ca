// Purpose: Verify a file request without AGENTS.md survives malformed decomposition and reaches approved execution.
// Pattern: HTTP, filesystem and session integration test.
// Usage: go test ./internal/tui/session -run TestReadmeRequest.
// Related: internal/tui/session/tasks.go, internal/agent/run/conversation.go, internal/tools/local/instructions.go.
package session

import (
	"context"
	"encoding/json"
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
		var request struct{ Messages []modelapi.Message }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		question := request.Messages[len(request.Messages)-1].Content
		reply := `{"tool":"write_file"}`
		switch {
		case strings.HasPrefix(question, "CONCLAVE REVIEW "):
			reply = "1"
		case strings.Contains(question, "\nGOAL ANALYSIS:"):
			reply = "Unable to format analysis"
		case strings.Contains(question, "\nSELF-GROUNDING:"):
			reply = "- Evidence: User requested README.md; observations are recorded.\n- Action: Perform the requested candidate.\n- Verify: Check the file after creation."
		case strings.Contains(question, "\nPFC PURPOSE:"):
			reply = "check"
		case strings.Contains(question, "ARGUMENT write_file.path"):
			reply = "\n\nREADME.md\n"
		case strings.Contains(question, "ARGUMENT write_file.content"):
			reply = "# Project\n"
		case strings.Contains(question, "ARGUMENT execute_command.command"):
			reply = "test -s README.md"
		case strings.Contains(question, `"Evidence":"execute_command`):
			reply = "DONE\nCreated README.md"
		case strings.Contains(question, "write_file result:"):
			reply = `{"tool":"execute_command"}`
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
	a.startTask("Create README.md in the current directory", nil)
	drainTask(t, a, true)
	content, err := os.ReadFile(filepath.Join(a.opts.Session.ProjectRoot, "README.md"))
	if err != nil || string(content) != "# Project\n" || a.status != "" {
		t.Fatalf("content=%q status=%q err=%v", content, a.status, err)
	}
}
