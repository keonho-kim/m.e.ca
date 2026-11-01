// Purpose: Verify isolated native LAT work and usage survive SQLite session persistence.
// Pattern: Scripted integration test over local tools and storage.
// Usage: go test ./internal/agent/session -run TestLATPersists.
// Related: internal/agent/run/lat.go, internal/storage/sqlite/events.go, internal/tools/local/read_only.go.
package session_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

type sessionLATModel struct {
	mu                sync.Mutex
	created, released int
	requests          map[string][]modelapi.Request
}

func (*sessionLATModel) Generate(context.Context, modelapi.Request, func(modelapi.Event) error) error {
	return fmt.Errorf("unscoped LAT generation")
}

func (m *sessionLATModel) NewRunGenerator() (modelapi.Generator, func()) {
	m.mu.Lock()
	m.created++
	m.mu.Unlock()
	phase := ""
	g := generation(func(_ context.Context, request modelapi.Request, send func(modelapi.Event) error) error {
		current := sessionLATPhase(request)
		if phase == "" {
			phase = current
		} else if phase != current {
			return fmt.Errorf("one scope served %s and %s", phase, current)
		}
		m.mu.Lock()
		m.requests[phase] = append(m.requests[phase], request)
		step := len(m.requests[phase])
		m.mu.Unlock()
		usage := &modelapi.TokenUsage{Input: 100, Output: 5, Cost: new(0.001)}
		if phase == "main" {
			switch step {
			case 1, 3:
				content := "one\n"
				if step == 3 {
					content = "two\n"
				}
				return send(sessionLATCall("write_file", fmt.Sprintf("main_%d", step), map[string]string{"path": "note.txt", "content": content}, usage))
			case 2:
				return send(modelapi.Event{Delta: "Initial draft", Thinking: "PRIVATE_MAIN_THINKING", Done: true, Usage: usage})
			case 4:
				return send(modelapi.Event{Delta: "Corrected file", Done: true, Usage: usage})
			}
			return fmt.Errorf("main script exhausted")
		}
		if step == 1 {
			file := "source.txt"
			if phase == "review" {
				file = "note.txt"
			}
			return send(sessionLATCall("execute_command", phase+"_read", map[string]string{"command": "cat " + file}, usage))
		}
		if step != 2 {
			return fmt.Errorf("%s script exhausted", phase)
		}
		text := map[string]string{"plan": "INITIAL_PLAN", "critique": "PLAN_CRITIQUE", "refine": "IMPROVED_PLAN", "review": "- note.txt: requested value is two, but observed content is one. Replace it with two."}[phase]
		return send(modelapi.Event{Delta: text, Done: true, Usage: usage})
	})
	return g, func() { m.mu.Lock(); defer m.mu.Unlock(); m.released++ }
}

func sessionLATCall(name, id string, args map[string]string, usage *modelapi.TokenUsage) modelapi.Event {
	body, _ := json.Marshal(args)
	return modelapi.Event{Calls: []toolapi.Call{{ID: id, Name: name, Arguments: body}}, Done: true, Usage: usage}
}

func sessionLATPhase(request modelapi.Request) string {
	for _, message := range request.Messages {
		for _, phase := range []string{"plan", "critique", "refine", "review"} {
			if strings.Contains(message.Content, "SESSION_LAT_"+strings.ToUpper(phase)) {
				return phase
			}
		}
	}
	return "main"
}

func TestLATPersistsOneConversationAndNativePhaseAudit(t *testing.T) {
	ctx := context.Background()
	project, home, state := t.TempDir(), t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "source.txt"), []byte("SOURCE_EVIDENCE\n"), 0600); err != nil {
		t.Fatal(err)
	}
	sessions, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := sessions.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	vault := &credentials.Vault{State: state}
	tools, err := local.NewExecutor(project, home, state, vault.Redact)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tools.Close() })
	model := &sessionLATModel{requests: map[string][]modelapi.Request{}}
	promptReads := 0
	request := agentrequest.New(agentrequest.Options{
		LAT: true, Config: config.Defaults(), Selection: connection.Selection{Model: "selected", Connection: connection.Connection{ID: "provider"}},
		Session: store.Session(), Generator: model, Tools: tools, Store: store, Vault: vault, Text: "note.txt에 two를 작성해주세요",
		LoadPrompts: func() (prompt.Templates, error) { return prompt.Defaults(), nil },
		LoadLAT: func() (prompt.Templates, error) {
			promptReads++
			value := prompt.Defaults()
			value.LATPlan += "\nSESSION_LAT_PLAN"
			value.LATCritique += "\nSESSION_LAT_CRITIQUE"
			value.LATRefine += "\nSESSION_LAT_REFINE"
			value.LAT += "\nSESSION_LAT_REVIEW"
			return value, nil
		},
	})
	result, err := session.Execute(ctx, request, func(context.Context, toolapi.Approval) (bool, error) { return true, nil }, func(runrecord.Event) error { return nil })
	if err != nil || result.Status != runrecord.Answered || result.Text != "Corrected file" || promptReads != 1 || model.created != 5 || model.released != 5 {
		t.Fatalf("LAT execution: %+v scopes=%d/%d prompt reads=%d err=%v", result, model.created, model.released, promptReads, err)
	}
	body, err := os.ReadFile(filepath.Join(project, "note.txt"))
	if err != nil || string(body) != "two\n" {
		t.Fatalf("correction was not applied: %q %v", body, err)
	}
	for _, phase := range []string{"plan", "critique", "refine", "review"} {
		requests := model.requests[phase]
		if len(requests) != 2 || !sessionNativePair(requests[1].Messages) {
			t.Fatalf("%s lost its native query/result: %+v", phase, requests)
		}
		for _, definition := range requests[0].Tools {
			if definition.Name == "write_file" || definition.Name == "edit_file" {
				t.Fatalf("%s could mutate files", phase)
			}
		}
	}
	usage, err := store.TokenUsage(ctx, store.Session().ID)
	if err != nil || usage.Input != 1200 || usage.Output != 60 || usage.Cost == nil || math.Abs(*usage.Cost-0.012) > 1e-12 {
		t.Fatalf("session usage: %+v %v", usage, err)
	}
	messages, err := store.ContextMessages(ctx, store.Session().ID, 0, 16*1024*1024, false)
	if err != nil {
		t.Fatal(err)
	}
	paired, users, improved := 0, 0, 0
	for i, message := range messages {
		if len(message.Calls) != 0 && i+1 < len(messages) && messages[i+1].ToolCallID == message.Calls[0].ID {
			paired++
		}
		if message.Role == "user" && message.Content == request.Text {
			users++
		}
		if strings.Contains(message.Content, "IMPROVED_PLAN") {
			improved++
		}
		if strings.Contains(message.Content, "INITIAL_PLAN") || strings.Contains(message.Content, "PLAN_CRITIQUE") || strings.Contains(message.Content, "SOURCE_EVIDENCE") {
			t.Fatal("private native history polluted parent context")
		}
	}
	if paired != 2 || users != 1 || improved != 1 {
		t.Fatalf("parent history: native pairs=%d original requests=%d improved plans=%d", paired, users, improved)
	}
	var audit strings.Builder
	if err := store.Export(ctx, store.Session().ID, &audit); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{`"phase":"plan"`, `"phase":"critique"`, `"phase":"refine"`, `"phase":"review"`, `"tool_call_id":"plan_read"`, "SOURCE_EVIDENCE"} {
		if !strings.Contains(audit.String(), value) {
			t.Errorf("audit missing %s", value)
		}
	}
}

func sessionNativePair(messages []modelapi.Message) bool {
	for i, message := range messages {
		if len(message.Calls) == 1 && i+1 < len(messages) && messages[i+1].ToolCallID == message.Calls[0].ID {
			return true
		}
	}
	return false
}
