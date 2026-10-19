// Purpose: Verify the built CLI bootstraps temporary state, writes through native calls and exports its session.
// Pattern: Executable integration test with a local provider fixture.
// Usage: go test ./cmd/v -run TestHeadlessExecutable.
// Related: cmd/v/headless.go, internal/commands/startup/parse.go.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHeadlessExecutable(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the executable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "v")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %s %v", out, err)
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		var input struct {
			Messages []struct{ Role, Content string }
			Tools    []struct {
				Function struct {
					Parameters struct{ Properties map[string]json.RawMessage }
				}
			}
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
			return
		}
		for _, tool := range input.Tools {
			if _, exists := tool.Function.Parameters.Properties["reason"]; exists {
				t.Error("obsolete reason schema")
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if calls.Load() == 1 {
			fmt.Fprint(w, `{"choices":[{"message":{"tool_calls":[{"id":"write","type":"function","function":{"name":"write_file","arguments":"{\"path\":\"README.md\",\"content\":\"# Example\\n\"}"}}]},"finish_reason":"tool_calls"}]}`)
		} else {
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
	cfg.Model = &config.SavedModel{Provider: connection.OpenAICompatible, Endpoint: server.URL, Name: "unknown-serving-alias"}
	if err := config.Save(filepath.Join(home, ".v"), cfg); err != nil {
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
	dbs, err := filepath.Glob(filepath.Join(home, ".v", "sessions", "*", "*.db"))
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
	if !strings.Contains(string(body), `"tool_call_id":"write"`) {
		t.Fatal("export lost the native result")
	}
	for _, name := range []string{"system.md", "development.md", "compaction.md"} {
		if _, err := os.Stat(filepath.Join(home, ".v", "prompts", name)); err != nil {
			t.Fatal(err)
		}
	}
}
