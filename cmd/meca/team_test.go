// Purpose: Verify native team collaboration against real isolated stores and local file tools.
// Pattern: Scripted model integration test.
// Usage: go test -race ./cmd/meca -run TestTeamSessions.
// Related: cmd/meca/team.go, internal/commands/teamrequest/request.go.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	teamcatalog "github.com/keonho-kim/meca/internal/tools/team/catalog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
)

type teamModel struct {
	mu      sync.Mutex
	steps   map[string]int
	restore bool
	sawPeer bool
}

func (g *teamModel) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	g.mu.Lock()
	id := ""
	for _, name := range []string{"agent-a", "agent-b", "agent-c"} {
		if strings.Contains(request.Messages[0].Content, "Your peer name is "+name+".") {
			id = name
		}
	}
	if id == "" {
		g.mu.Unlock()
		return fmt.Errorf("missing peer identity")
	}
	for _, m := range request.Messages {
		if strings.Contains(m.Content, "Public conversation — agent-") && !strings.Contains(m.Content, "Public conversation — "+id+"\n") {
			g.sawPeer = true
		}
	}
	step := g.steps[id]
	g.steps[id]++
	restore := g.restore
	g.mu.Unlock()
	var name string
	var arguments any
	switch step {
	case 0:
		if id == "agent-a" {
			name = "call_friend"
			arguments = map[string]string{"message": "Create agent-c.txt with an independent contribution. Keep agent-a.txt and agent-b.txt unchanged and report your result publicly."}
			if restore {
				arguments.(map[string]string)["agent_id"] = "agent-c"
			}
		} else {
			name = "list_friends"
			arguments = map[string]string{}
		}
	case 1:
		name = "write_file"
		arguments = map[string]string{"path": id + ".txt", "content": "A verified local contribution\n"}
		if id == "agent-b" {
			name = "execute_command"
			arguments = map[string]string{"command": "printf 'A verified local contribution\\n' > agent-b.txt"}
		}
	case 2:
		name = "say_goodbye"
		arguments = map[string]string{"message": id + " created its file. No integration test was requested."}
	default:
		if err := emit(modelapi.Event{Streaming: true, Delta: "The requested contributions are written."}); err != nil {
			return err
		}
		return emit(modelapi.Event{Done: true, Usage: &modelapi.TokenUsage{Input: 11, Output: 7}})
	}
	body, _ := json.Marshal(arguments)
	call := toolapi.Call{ID: fmt.Sprintf("%s-%d", id, step), Name: name, Arguments: body}
	if err := ctx.Err(); err != nil {
		return err
	}
	return emit(modelapi.Event{Done: true, Delta: "I will update " + id + ".txt with my independent contribution.", Calls: []toolapi.Call{call}, Usage: &modelapi.TokenUsage{Input: 11, Output: 7}})
}

func TestTeamSessionsExecuteAndRestoreWithPrivateDatabases(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	configuration, err := bootstrap.Initialize(home)
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(home, ".meca")
	catalog, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := catalog.Create(context.Background(), project, project)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	vault := &credentials.Vault{State: state}
	service := teamService(store, home, state, vault)
	local, err := service.NewTools()
	if err != nil {
		t.Fatal(err)
	}
	for _, definition := range local.Definitions() {
		if teamcatalog.IsTool(definition.Name) {
			t.Fatalf("team tool leaked into the single executor: %s", definition.Name)
		}
	}
	if err := local.Close(); err != nil {
		t.Fatal(err)
	}
	var original []string
	for round := range 2 {
		model := &teamModel{steps: map[string]int{}, restore: round > 0}
		var events []runrecord.Event
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		result, err := service.Run(ctx, agentrequest.Options{Config: configuration, Session: store.Session(), Selection: connection.Selection{Model: "test-model"}, Generator: model, Vault: vault, Text: "Create independent contributions inside the project."}, func(context.Context, toolapi.Approval) (bool, error) { return true, nil }, func(event runrecord.Event) error { events = append(events, event); return nil })
		cancel()
		if err != nil || result.Status != runrecord.Answered || result.Text == "" {
			t.Fatalf("round %d result=%+v error=%v", round, result, err)
		}
		for _, event := range events {
			if event.Kind == "tool_failed" || event.Kind == "tool_interrupted" {
				t.Fatalf("scripted happy path failed a tool: %+v", event)
			}
		}
		members, err := store.TeamMembers(context.Background(), "", 100)
		if err != nil || len(members) != 3 {
			t.Fatalf("members=%+v err=%v", members, err)
		}
		if round == 0 {
			for _, m := range members {
				original = append(original, m.Session)
			}
		} else {
			for i, m := range members {
				if original[i] != m.Session {
					t.Fatal("restored a different private session")
				}
			}
		}
		if !model.sawPeer {
			t.Fatal("public speech never reached private contexts")
		}
		for _, member := range members {
			body, err := os.ReadFile(filepath.Join(project, member.ID+".txt"))
			if err != nil || string(body) != "A verified local contribution\n" {
				t.Fatalf("file %s: %q %v", member.ID, body, err)
			}
			private, err := store.OpenTeamAgent(context.Background(), member.Session)
			if err != nil {
				t.Fatal(err)
			}
			cursor, err := private.ContextCursor(context.Background(), private.Session().ID)
			if closeErr := private.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
			if err != nil || cursor == 0 {
				t.Fatalf("missing durable receive cursor: %d %v", cursor, err)
			}
		}
		if len(service.CommandActivity()) != 0 {
			t.Fatal("executor retained after team completion")
		}
	}
	ctx := context.Background()
	records, _, err := store.Timeline(ctx, store.Session().ID, 0, 200)
	if err != nil {
		t.Fatal(err)
	}
	resolved := false
	for _, record := range records {
		resolved = resolved || record.Event.Agent != "" && record.Event.Call != nil && record.Event.Call.Name == "write_file"
	}
	if !resolved {
		t.Fatal("private tool references did not restore")
	}
	var exported strings.Builder
	if err := store.Export(ctx, store.Session().ID, &exported); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(exported.String(), `"kind":"team_reference"`) {
		t.Fatal("export did not hydrate original referenced events")
	}
	conversations, err := catalog.Conversations(ctx, project)
	if err != nil || len(conversations) != 1 {
		t.Fatalf("private sessions leaked into conversation picker: %d %v", len(conversations), err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := catalog.Open(ctx, project, conversations[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	members, err := reopened.TeamMembers(ctx, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range members {
		private, err := reopened.OpenTeamAgent(ctx, member.Session)
		if err != nil {
			t.Fatal(err)
		}
		messages, err := private.ContextMessages(ctx, private.Session().ID, 0, 1<<20, false)
		closeErr := private.Close()
		if err != nil || closeErr != nil || len(messages) == 0 {
			t.Fatalf("private context did not survive reopening: %v %v", err, closeErr)
		}
	}
}
