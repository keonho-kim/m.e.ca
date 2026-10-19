// Purpose: Verify print and JSON modes execute the shared loop and preserve approval boundaries.
// Pattern: Native tool integration tests with a scripted generator.
// Usage: go test ./internal/commands/headless.
// Related: internal/commands/headless/run.go, internal/agent/session/request.go.
package headless

import (
	"bytes"
	"context"
	"encoding/json"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"github.com/keonho-kim/v-code/internal/agent/run"
	sessionrun "github.com/keonho-kim/v-code/internal/agent/session"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	"github.com/keonho-kim/v-code/internal/tools/local"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type script struct{ calls int }

func (g *script) Generate(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
	g.calls++
	if g.calls == 1 {
		return emit(modelapi.Event{Done: true, Calls: []toolapi.Call{{ID: "write", Name: "write_file", Arguments: json.RawMessage(`{"path":"hello.txt","content":"hello\n"}`)}}})
	}
	if err := emit(modelapi.Event{Streaming: true, Delta: "Created "}); err != nil {
		return err
	}
	return emit(modelapi.Event{Streaming: true, Delta: "hello.txt", Done: true})
}

func TestSharedLoopPrintJSONAndApproval(t *testing.T) {
	for _, tc := range []struct {
		name          string
		json, approve bool
	}{{"print", false, false}, {"json", true, false}, {"approval", true, true}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			home := t.TempDir()
			project := filepath.Join(home, "project")
			if err := os.Mkdir(project, 0700); err != nil {
				t.Fatal(err)
			}
			sessions, err := sqlite.OpenSessions(filepath.Join(home, ".v", "sessions"))
			if err != nil {
				t.Fatal(err)
			}
			store, err := sessions.Create(ctx, project, project)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { store.Close() })
			executor, err := local.NewExecutor(project, home, filepath.Join(home, ".v"), func(s string) string { return s })
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { executor.Close() })
			settings := policy.Settings{Scope: policy.System, Permission: policy.AllowAll}
			if tc.approve {
				settings.Permission = policy.Approve
			}
			cfg := config.Defaults()
			g := &script{}
			runner := run.Runner{Generator: g, Journal: store, Tools: executor, Session: store.Session().ID, Model: "fixture", Settings: settings, Redact: func(s string) string { return s }}
			var output bytes.Buffer
			err = Run(ctx, sessionrun.Request{Runner: runner, Store: store, Text: "Create hello.txt", Compaction: cfg.Compaction.Settings, ProjectInstructions: executor.ProjectInstructions, LoadPrompts: func() (prompt.Templates, error) { return prompt.Defaults(), nil }}, &output, tc.json)
			if tc.approve {
				if err == nil {
					t.Fatal("noninteractive approval was bypassed")
				}
				if _, statErr := os.Stat(filepath.Join(project, "hello.txt")); !os.IsNotExist(statErr) {
					t.Fatal("unapproved file exists")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				body, err := os.ReadFile(filepath.Join(project, "hello.txt"))
				if err != nil || string(body) != "hello\n" {
					t.Fatalf("file=%q %v", body, err)
				}
				if !tc.json && output.String() != "Created hello.txt\n" {
					t.Fatalf("unexpected print %q", output.String())
				}
			}
			if tc.json {
				for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
					if !json.Valid([]byte(line)) {
						t.Fatalf("non-JSON stdout %q", line)
					}
				}
				if !strings.Contains(output.String(), `"kind":"run_finished"`) {
					t.Fatal("missing terminal event")
				}
			}
		})
	}
}
