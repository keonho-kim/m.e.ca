// Purpose: Verify the CLI restores credentials, executes a code-driven write and exports its results.
// Pattern: Executable integration test with a local provider fixture.
// Usage: go test ./cmd/meca -run TestHeadlessExecutable.
// Related: cmd/meca/headless.go, internal/config/credentials/persistence.go, internal/commands/startup/parse.go.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/modeltest"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestHeadlessExecutable(t *testing.T) {
	binary := testExecutable(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	const secret = "fixture-cli-api-key"
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+secret {
			t.Error("saved local credential was not restored by the CLI")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		var input struct {
			Model    string
			Messages []modelapi.Message
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
			http.Error(w, "invalid generation request", http.StatusBadRequest)
			return
		}
		if input.Model != "unknown-serving-alias" {
			t.Errorf("requested model = %q", input.Model)
		}
		w.Header().Set("Content-Type", "application/json")
		if calls.Load() == 1 {
			tools := modeltest.ChatCalls([]toolapi.Call{{ID: "write", Name: "write_file", Arguments: json.RawMessage(`{"reason":"Document the project","path":"README.md","content":"# Example\n"}`)}})
			if err := json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"tool_calls": tools}, "finish_reason": "tool_calls"}}}); err != nil {
				t.Error(err)
			}
		} else {
			completed := false
			for _, message := range input.Messages {
				if message.Role == "tool" && message.ToolCallID == "write" {
					var result struct{ Status string }
					completed = json.Unmarshal([]byte(message.Content), &result) == nil && result.Status == "completed"
				}
			}
			if !completed {
				t.Error("model did not receive the completed file-write result")
				http.Error(w, "missing completed file-write result", http.StatusBadRequest)
				return
			}
			fmt.Fprint(w, `{"choices":[{"message":{"content":"Created README.md"},"finish_reason":"stop"}]}`)
		}
	}))
	defer server.Close()
	home := t.TempDir()
	project := filepath.Join(home, "project")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.Model = &config.SavedModel{Provider: connection.OpenRouter, Endpoint: server.URL, Name: "unknown-serving-alias", CredentialRef: "credential_cli"}
	state := filepath.Join(home, ".meca")
	conn, err := connection.New(connection.OpenRouter, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := (&credentials.Vault{State: state}).Set(ctx, cfg.Model.CredentialRef, conn.CredentialScope(), secret); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(state, cfg); err != nil {
		t.Fatal(err)
	}
	run := exec.CommandContext(ctx, binary, project, "--mode", "json", "Create README.md")
	run.Env = append(os.Environ(), "HOME="+home)
	body, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run: %s %v", body, err)
	}
	if calls.Load() != 2 {
		t.Fatalf("unexpected setup generations: %d", calls.Load())
	}
	for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		if !json.Valid([]byte(line)) {
			t.Fatalf("non-JSON stdout: %q", line)
		}
	}
	content, err := os.ReadFile(filepath.Join(project, "README.md"))
	if err != nil || string(content) != "# Example\n" {
		t.Fatalf("file=%q %v", content, err)
	}
	dbs, err := filepath.Glob(filepath.Join(home, ".meca", "sessions", "*", "*.db"))
	if err != nil || len(dbs) != 1 {
		t.Fatalf("dbs=%v %v", dbs, err)
	}
	id := strings.TrimSuffix(filepath.Base(dbs[0]), ".db")
	export := exec.CommandContext(ctx, binary, project, "--session", id, "--export")
	export.Env = run.Env
	body, err = export.CombinedOutput()
	if err != nil {
		t.Fatalf("export: %s %v", body, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	codeResult, fileResult := false, false
	for {
		var record struct {
			Kind string
			Body json.RawMessage
		}
		if err := decoder.Decode(&record); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		switch record.Kind {
		case "message":
			var message modelapi.Message
			if err := json.Unmarshal(record.Body, &message); err != nil {
				t.Fatal(err)
			}
			codeResult = codeResult || message.Role == "tool" && message.ToolCallID == "write" && message.ToolName == "codemode"
		case "event":
			var event runrecord.Event
			if err := json.Unmarshal(record.Body, &event); err != nil {
				t.Fatal(err)
			}
			fileResult = fileResult || event.Kind == "tool_succeeded" && event.Call != nil && event.Call.Name == "write_file" && event.Call.ParentExecutionID != "" && len(event.Output) != 0
		}
	}
	if !codeResult || !fileResult {
		t.Fatalf("export lost completed results: code=%v file=%v", codeResult, fileResult)
	}
}
