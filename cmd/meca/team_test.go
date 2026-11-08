// Purpose: Verify native team collaboration against real isolated stores and local file tools.
// Pattern: Scripted model integration test.
// Usage: go test -race ./cmd/meca -run TestTeamSessions.
// Related: cmd/meca/team.go, internal/commands/teamrequest/request.go.
package main

import (
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	teamcatalog "github.com/keonho-kim/meca/internal/tools/team/catalog"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/commands/teamrequest"
	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
)

type teamModel struct {
	mu      sync.Mutex
	steps   map[string]int
	names   []string
	restore bool
	sawPeer bool
}

func (g *teamModel) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	g.mu.Lock()
	id := ""
	for _, name := range g.names {
		if strings.Contains(request.Messages[0].Content, "Your peer name is "+name+".") {
			id = name
		}
	}
	if id == "" {
		_, tail, found := strings.Cut(request.Messages[0].Content, "Your peer name is ")
		if !found {
			g.mu.Unlock()
			return fmt.Errorf("missing peer identity")
		}
		id, _, _ = strings.Cut(tail, ".")
		g.names = append(g.names, id)
	}
	for _, m := range request.Messages {
		if m.Role == "user" && strings.HasPrefix(m.Content, "Public conversation — ") {
			speaker := strings.TrimPrefix(m.Content, "Public conversation — ")
			if end := strings.IndexAny(speaker, " \t\r\n"); end >= 0 {
				speaker = speaker[:end]
			}
			if speaker != id {
				g.sawPeer = true
			}
		}
	}
	step := g.steps[id]
	g.steps[id]++
	restore := g.restore
	names := append([]string(nil), g.names...)
	g.mu.Unlock()
	var name string
	var arguments any
	retryDeparture := false
	for i := len(request.Messages) - 1; i >= 0; i-- {
		m := request.Messages[i]
		if m.Role != "tool" {
			continue
		}
		if m.ToolName == "leave_team" {
			var result struct{ Failure *faults.Failure }
			if json.Unmarshal([]byte(m.Content), &result) == nil && result.Failure != nil && result.Failure.Code == "team_feedback_pending" {
				retryDeparture = true
			}
		}
		break
	}
	if retryDeparture {
		name = "leave_team"
		arguments = map[string]string{"message": id + " reviewed the current teammate results and finished its contribution."}
	} else {
		switch step {
		case 0:
			if id == names[0] {
				name = "add_team_member"
				arguments = map[string]string{"message": "Create your own name.txt as an independent contribution. Use set_work first. Preserve the other teammates files and report results publicly."}
				if restore {
					arguments.(map[string]string)["member_name"] = names[2]
				}
			} else {
				name = "list_team_members"
				arguments = map[string]string{}
			}
		case 1:
			name = "set_work"
			arguments = map[string]any{"summary": "Create my independent file", "paths": []string{id + ".txt"}}
		case 2:
			name = "write_file"
			arguments = map[string]string{"path": id + ".txt", "content": "A verified local contribution\n"}
			if id == names[1] {
				name = "execute_command"
				arguments = map[string]string{"command": "printf 'A verified local contribution\\n' > " + id + ".txt"}
			}
		case 3:
			name = "leave_team"
			arguments = map[string]string{"message": id + " created its file. No integration test was requested."}
		default:
			if err := emit(modelapi.Event{Streaming: true, Delta: "The requested contributions are written."}); err != nil {
				return err
			}
			return emit(modelapi.Event{Done: true, Usage: &modelapi.TokenUsage{Input: 11, Output: 7}})
		}
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
	local, err := service.NewTools(teamrequest.LocalOptions{})
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
	for range 2 {
		if _, err := store.CreateTeamAgent(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	for round := range 2 {
		seeded, err := store.TeamMembers(context.Background(), "", 8)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, member := range seeded {
			names = append(names, member.ID)
		}
		model := &teamModel{steps: map[string]int{}, names: names, restore: round > 0}
		var events []runrecord.Event
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		result, err := service.Run(ctx, agentrequest.Options{Runtime: agentrequest.Runtime{Config: configuration, Session: store.Session(), Selection: connection.Selection{Model: "test-model"}, Vault: vault}, Text: "Create independent contributions inside the project.", Generator: model}, func(context.Context, toolapi.Approval) (bool, error) { return true, nil }, func(event runrecord.Event) error { events = append(events, event); return nil })
		cancel()
		if err != nil || result.Status != runrecord.Answered || result.Text == "" {
			t.Fatalf("round %d result=%+v error=%v", round, result, err)
		}
		sawShellTargets := false
		deferredDepartures := map[string]bool{}
		for _, event := range events {
			sawShellTargets = sawShellTargets || event.Kind == "team_observation" && strings.Contains(event.Text, "May have affected:")
			if event.Kind == "tool_failed" || event.Kind == "tool_interrupted" {
				if op := event.Operation; event.Kind == "tool_failed" && event.Call != nil && event.Call.Name == "leave_team" && op != nil && op.Failure != nil && op.Failure.Code == "team_feedback_pending" {
					deferredDepartures[event.Agent] = true
					continue
				}
				t.Fatalf("scripted happy path failed a tool: %+v", event)
			}
			if event.Kind == "tool_succeeded" && event.Call != nil && event.Call.Name == "leave_team" {
				delete(deferredDepartures, event.Agent)
			}
		}
		if len(deferredDepartures) > 0 {
			t.Fatal("deferred departure never reviewed the feedback", deferredDepartures)
		}
		if !sawShellTargets {
			t.Fatal("known shell mutation targets were not shared as uncertain effects")
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
		messages, err := private.ContextMessages(ctx, private.Session().ID, 0, 1<<20, false, nil)
		closeErr := private.Close()
		if err != nil || closeErr != nil || len(messages) == 0 {
			t.Fatalf("private context did not survive reopening: %v %v", err, closeErr)
		}
	}
}
