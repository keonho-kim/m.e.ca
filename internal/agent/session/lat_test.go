// Purpose: Verify LAT repairs real tool work in one session and persists measured review usage.
// Pattern: Integration test over local files and SQLite with a scripted model.
// Usage: go test ./internal/agent/session -run TestLAT.
// Related: internal/commands/agentrequest/request.go, internal/agent/run/lat.go, internal/storage/sqlite/usage.go.
package session_test

import (
	"context"
	"encoding/json"
	"math"
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

func TestLATPersistsOneConversationAndReviewUsage(t *testing.T) {
	ctx := context.Background()
	project, home, state := t.TempDir(), t.TempDir(), t.TempDir()
	sessions, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
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
	vault := &credentials.Vault{State: state}
	tools, err := local.NewExecutor(project, home, state, vault.Redact)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tools.Close(); err != nil {
			t.Error(err)
		}
	})
	step, reviews, promptReads, summaries := 0, 0, 0, 0
	g := generation(func(_ context.Context, request modelapi.Request, send func(modelapi.Event) error) error {
		usage := &modelapi.TokenUsage{Input: 100, Output: 10, Cost: new(0.001)}
		if strings.HasPrefix(request.Messages[0].Content, "# Summarize work for continuation") {
			summaries++
			if !strings.Contains(request.Messages[1].Content, "requested value is two") {
				t.Fatal("compaction dropped the unresolved review finding")
			}
			return send(modelapi.Event{Delta: "- Pending review correction: note.txt must contain two; observed content is one.", Done: true, Usage: usage})
		}
		if len(request.Tools) == 0 {
			reviews++
			if !strings.Contains(request.Messages[0].Content, "CUSTOM_LAT_RULE") {
				t.Error("installed LAT prompt was not used")
			}
			return send(modelapi.Event{Delta: "- note.txt: requested value is two, but observed content is one. Replace it with two.", Done: true, Usage: usage})
		}
		step++
		if step == 3 {
			return &modelapi.ContextLimitError{}
		}
		if step == 2 || step == 5 {
			return send(modelapi.Event{Delta: "파일을 작성했습니다.", Done: true, Usage: usage})
		}
		content := "one\n"
		if step == 4 {
			found := false
			for _, message := range request.Messages {
				found = found || strings.Contains(message.Content, "Pending review correction")
			}
			if !found {
				t.Fatal("main agent lost pending feedback after compaction")
			}
			content = "two\n"
		}
		arguments, _ := json.Marshal(map[string]string{"path": "note.txt", "content": content})
		return send(modelapi.Event{Calls: []toolapi.Call{{ID: "write-" + content[:3], Name: "write_file", Arguments: arguments}}, Done: true, Usage: usage})
	})
	selected := connection.Selection{Model: "selected", Connection: connection.Connection{ID: "provider"}}
	cfg := config.Defaults()
	cfg.Compaction.KeepRecentTokens = 0 // Exercise feedback preservation through the real history projection.
	request := agentrequest.New(agentrequest.Options{LAT: true, Config: cfg, Selection: selected, Session: store.Session(), Generator: g, Tools: tools, Store: store, Vault: vault,
		Text: "note.txt에 two를 작성해주세요", LoadPrompts: func() (prompt.Templates, error) { return prompt.Defaults(), nil },
		LoadLAT: func() (prompt.Templates, error) {
			promptReads++
			value := prompt.Defaults()
			value.LAT += "\nCUSTOM_LAT_RULE"
			return value, nil
		},
	})
	result, err := session.Execute(ctx, request, func(context.Context, toolapi.Approval) (bool, error) { return true, nil }, func(runrecord.Event) error { return nil })
	if err != nil || result.Status != runrecord.Answered || step != 5 || reviews != 1 || promptReads != 1 || summaries != 1 {
		t.Fatalf("LAT execution: %+v steps=%d reviews=%d summaries=%d error=%v", result, step, reviews, summaries, err)
	}
	body, err := os.ReadFile(filepath.Join(project, "note.txt"))
	if err != nil || string(body) != "two\n" {
		t.Fatalf("correction was not applied: %q %v", body, err)
	}
	usage, err := store.TokenUsage(ctx, store.Session().ID)
	if err != nil || usage.Input != 600 || usage.Output != 60 || usage.Cost == nil || math.Abs(*usage.Cost-0.006) > 1e-12 {
		t.Fatalf("session usage: %+v %v", usage, err)
	}
	messages, err := store.ContextMessages(ctx, store.Session().ID, 0, 16*1024*1024, false)
	if err != nil {
		t.Fatal(err)
	}
	paired, feedback := 0, 0
	for i, message := range messages {
		if len(message.Calls) != 0 && i+1 < len(messages) && messages[i+1].ToolCallID == message.Calls[0].ID {
			paired++
		}
		if message.Role == "system" && strings.Contains(message.Content, "requested value is two") {
			feedback++
		}
	}
	if paired != 2 || feedback != 1 {
		t.Fatalf("native pairing or review context lost: calls=%d feedback=%d", paired, feedback)
	}
}
