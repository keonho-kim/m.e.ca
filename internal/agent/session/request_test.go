// Purpose: Verify one main agent executes tools and retains follow-up history in its session.
// Pattern: Scripted model integration test over real file and persistence adapters.
// Usage: go test ./internal/agent/session -run TestMainAgentExecutesFollowUpDirectly.
// Related: internal/agent/session/request.go, internal/storage/sqlite/store.go, internal/tools/local/executor.go.
package session_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/agent/compact"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/agent/session"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	"github.com/keonho-kim/v-code/internal/tools/local"
)

type generation func(context.Context, modelapi.Request, func(modelapi.Event) error) error

func (g generation) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	return g(ctx, request, emit)
}

func TestMainAgentExecutesFollowUpDirectly(t *testing.T) {
	ctx := context.Background()
	project := t.TempDir()
	sessions, err := sqlite.OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := sessions.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	tools, err := local.NewExecutor(project, t.TempDir(), t.TempDir(), nil, func(s string) string { return s })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tools.Close(); err != nil {
			t.Error(err)
		}
	})
	templates := prompt.Defaults()
	templates.Development = "PROJECT_DEVELOPMENT_RULE"
	step, approvals := 0, 0
	g := generation(func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		catalog := map[string]bool{}
		for _, definition := range request.Tools {
			catalog[definition.Name] = true
		}
		if len(catalog) != len(tools.Definitions()) {
			t.Fatalf("main agent must receive the full direct tool catalog: %v", catalog)
		}
		for _, definition := range tools.Definitions() {
			if !catalog[definition.Name] {
				t.Fatalf("main agent lost tool %s", definition.Name)
			}
		}
		if !strings.Contains(request.Messages[0].Content, "PROJECT_DEVELOPMENT_RULE") || !strings.Contains(request.Messages[0].Content, "PROJECT_INSTRUCTION") {
			t.Fatal("main agent lost development or project instructions")
		}
		if request.Temperature == nil || *request.Temperature != 1 {
			t.Fatal("main agent temperature changed")
		}
		step++
		if step == 2 || step == 4 {
			return emit(modelapi.Event{Delta: "변경했습니다.", Done: true})
		}
		name, args := "write_file", map[string]any{"path": "note.txt", "content": "one\n", "reason": "Create the requested note."}
		if step == 3 {
			var history strings.Builder
			for _, message := range request.Messages {
				history.WriteString(message.Content)
			}
			if !strings.Contains(history.String(), "Create note.txt") || !strings.Contains(history.String(), "Update note.txt") {
				t.Fatal("follow-up lost prior or current user request")
			}
			if !strings.Contains(request.Messages[len(request.Messages)-1].Content, `"state":"present"`) {
				t.Fatal("follow-up lost the current file observation")
			}
			name = "edit_file"
			args = map[string]any{"path": "note.txt", "edits": []any{map[string]any{"old_text": "one", "new_text": "two"}}, "reason": "Apply the requested follow-up to the existing note."}
		}
		body, err := json.Marshal(args)
		if err != nil {
			return err
		}
		return emit(modelapi.Event{Calls: []toolapi.Call{{ID: name + "-call", Name: name, Arguments: body}}, Done: true})
	})
	runner := run.Runner{Generator: g, Tools: tools, Journal: store, Session: store.Session().ID, ProjectRoot: project, Settings: policy.Settings{Permission: policy.Approve, Scope: policy.System}, Redact: func(s string) string { return s }}
	for _, text := range []string{"Create note.txt", "Update note.txt from one to two"} {
		result, err := session.Execute(ctx, session.Request{Runner: runner, Store: store, Text: text, LoadPrompts: func() (prompt.Templates, error) { return templates, nil }, Compaction: compact.Defaults(), ProjectInstructions: func(context.Context) (string, error) { return "PROJECT_INSTRUCTION", nil }}, func(context.Context, toolapi.Approval) (bool, error) {
			approvals++
			return true, nil
		}, func(run.Event) error { return nil })
		if err != nil || result.Status != run.Answered {
			t.Fatalf("direct execution failed: %+v %v", result, err)
		}
	}
	content, err := os.ReadFile(filepath.Join(project, "note.txt"))
	if err != nil || string(content) != "two\n" || approvals != 2 || step != 4 {
		t.Fatalf("content=%q approvals=%d calls=%d error=%v", content, approvals, step, err)
	}
	messages, err := store.ContextMessages(ctx, store.Session().ID, 0, 16*1024*1024)
	if err != nil || len(messages) != 8 {
		t.Fatalf("single conversation history: messages=%d error=%v", len(messages), err)
	}
}
