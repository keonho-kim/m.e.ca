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

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/session"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local"
)

type generation func(context.Context, modelapi.Request, func(modelapi.Event) error) error

func (g generation) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	return g(ctx, request, emit)
}

func TestMainAgentExecutesFollowUpDirectly(t *testing.T) {
	ctx := context.Background()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "AGENTS.md"), []byte("PROJECT_INSTRUCTION"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.Agent.Temperature, cfg.Agent.MaxOutputTokens = 0.73, 1234
	on, thinking := true, false
	cfg.Thinking = &on
	selected := connection.Selection{Model: "selected-model", Connection: connection.Connection{ID: "selected-provider"}, Thinking: &thinking, ThinkingSupported: true, Details: &connection.ModelInfo{ContextWindow: 131072}}
	vault := &credentials.Vault{State: t.TempDir()}
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
	tools, err := local.NewExecutor(project, t.TempDir(), t.TempDir(), func(s string) string { return s })
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
		if len(catalog) != len(tools.Definitions())-1 {
			t.Fatalf("main agent must receive the full direct tool catalog: %v", catalog)
		}
		for _, definition := range tools.Definitions() {
			if definition.Name != "read_image" && !catalog[definition.Name] {
				t.Fatalf("main agent lost tool %s", definition.Name)
			}
		}
		if !strings.Contains(request.Messages[0].Content, "PROJECT_DEVELOPMENT_RULE") || !strings.Contains(request.Messages[0].Content, "PROJECT_INSTRUCTION") {
			t.Fatal("main agent lost development or project instructions")
		}
		if request.Temperature == nil || *request.Temperature != 0.73 {
			t.Fatal("shared request factory lost configured temperature")
		}
		if request.ModelID != "selected-model" || request.MaxOutput != 1234 || request.Thinking == nil || *request.Thinking {
			t.Fatal("shared request factory lost selected model, output budget or Thinking OFF")
		}
		step++
		if step == 2 || step == 4 {
			return emit(modelapi.Event{Delta: "변경했습니다.", Done: true})
		}
		name, args := "write_file", map[string]any{"path": "note.txt", "content": "one\n"}
		if step == 3 {
			var history strings.Builder
			for _, message := range request.Messages {
				history.WriteString(message.Content)
			}
			if !strings.Contains(history.String(), "Create note.txt") || !strings.Contains(history.String(), "Update note.txt") {
				t.Fatal("follow-up lost prior or current user request")
			}
			if !strings.Contains(history.String(), "one") {
				t.Fatal("follow-up lost the current file observation")
			}
			name = "edit_file"
			args = map[string]any{"path": "note.txt", "edits": []any{map[string]any{"old_text": "one", "new_text": "two"}}}
		}
		body, err := json.Marshal(args)
		if err != nil {
			return err
		}
		return emit(modelapi.Event{Calls: []toolapi.Call{{ID: name + "-call", Name: name, Arguments: body}}, Done: true})
	})
	for _, text := range []string{"Create note.txt", "Update note.txt from one to two"} {
		thinking = false
		request := agentrequest.New(agentrequest.Options{Config: cfg, Selection: selected, Session: store.Session(), Generator: g, Tools: tools, Store: store, Vault: vault, Text: text, LoadPrompts: func() (prompt.Templates, error) { return templates, nil }})
		thinking = true // Mutating the selection after submission must not alter the running request.
		if request.ContextWindow != 131072 || request.Runner.ProjectRoot != store.Session().ProjectRoot {
			t.Fatal("shared request factory lost serving limits or the selected session root")
		}
		result, err := session.Execute(ctx, request, func(context.Context, toolapi.Approval) (bool, error) {
			approvals++
			return true, nil
		}, func(runrecord.Event) error { return nil })
		if err != nil || result.Status != runrecord.Answered {
			t.Fatalf("direct execution failed: %+v %v", result, err)
		}
	}
	content, err := os.ReadFile(filepath.Join(project, "note.txt"))
	if err != nil || string(content) != "two\n" || approvals != 0 || step != 4 {
		t.Fatalf("content=%q approvals=%d calls=%d error=%v", content, approvals, step, err)
	}
	messages, err := store.ContextMessages(ctx, store.Session().ID, 0, 16*1024*1024, true)
	if err != nil || len(messages) != 8 {
		t.Fatalf("single conversation history: messages=%d error=%v", len(messages), err)
	}
}
