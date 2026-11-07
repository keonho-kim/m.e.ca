// Purpose: Recover from overlapping contributions and share real file outcomes through full team execution.
// Pattern: Scripted two-member integration with real private databases and file tools.
// Usage: Command tests exercise coordination without model or network dependencies.
// Related: cmd/meca/team.go, internal/tools/team/work.go, internal/agent/team/observations.go.
package main

import (
	"context"
	"encoding/json"
	"fmt"
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

type workTeamModel struct {
	mu                         sync.Mutex
	names                      []string
	steps                      map[string]int
	claimed, conflict, changed chan struct{}
	sawOwner, sawResult        bool
}

func (g *workTeamModel) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	id := g.names[0]
	if strings.Contains(request.Messages[0].Content, "Your peer name is "+g.names[1]+".") {
		id = g.names[1]
	}
	g.mu.Lock()
	step := g.steps[id]
	g.steps[id]++
	if id == g.names[1] {
		for _, message := range request.Messages {
			g.sawOwner = g.sawOwner || strings.Contains(message.Content, "team_work_conflict") && strings.Contains(message.Content, g.names[0])
			g.sawResult = g.sawResult || strings.Contains(message.Content, "write_file tool_succeeded") && strings.Contains(message.Content, "shared.txt")
		}
	}
	g.mu.Unlock()
	wait := func(signal <-chan struct{}) error {
		select {
		case <-signal:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	name, text := "", ""
	var args any
	if id == g.names[0] {
		switch step {
		case 0:
			name, args = "set_work", map[string]any{"summary": "I will implement shared.txt; " + g.names[1] + " can handle independent checks.", "paths": []string{"shared.txt"}}
		case 1:
			if err := wait(g.conflict); err != nil {
				return err
			}
			name, args = "write_file", map[string]string{"path": "shared.txt", "content": "one implementation\n"}
		case 2:
			name, args = "leave_team", map[string]string{"message": "shared.txt is ready; preserve it and finish the independent checks."}
		}
	} else {
		switch step {
		case 0:
			if err := wait(g.claimed); err != nil {
				return err
			}
			name, args = "set_work", map[string]any{"summary": "Implement the same file", "paths": []string{"shared.txt"}}
		case 1:
			name, args = "set_work", map[string]any{"summary": g.names[0] + " owns shared.txt; I will handle checks.txt.", "paths": []string{"checks.txt"}}
		case 2:
			if err := wait(g.changed); err != nil {
				return err
			}
			name, args = "read_team_messages", map[string]any{}
		case 3:
			name, args = "write_file", map[string]string{"path": "checks.txt", "content": "independent checks\n"}
		case 4:
			name, args = "leave_team", map[string]string{"message": "checks.txt is complete; shared.txt was preserved."}
		}
	}
	if name == "" {
		return emit(modelapi.Event{Done: true, Delta: "Both contributions are complete; no additional checks were run."})
	}
	body, _ := json.Marshal(args)
	return emit(modelapi.Event{Done: true, Delta: text, Calls: []toolapi.Call{{ID: fmt.Sprintf("%s-%d", id, step), Name: name, Arguments: body}}})
}

func TestTeamRecoversFromDuplicateScopeAndSharesActualResult(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	cfg, err := bootstrap.Initialize(home)
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
	var names []string
	for range 2 {
		member, err := store.CreateTeamAgent(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, member.ID)
	}
	g := &workTeamModel{names: names, steps: map[string]int{}, claimed: make(chan struct{}), conflict: make(chan struct{}), changed: make(chan struct{})}
	vault := &credentials.Vault{State: state}
	service := teamService(store, home, state, vault)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	writes, conflicts := 0, 0
	result, err := service.Run(ctx, agentrequest.Options{Runtime: agentrequest.Runtime{Config: cfg, Session: store.Session(), Selection: connection.Selection{Model: "fixture"}, Vault: vault}, Text: "Implement shared.txt and independent checks.txt together.", Generator: g}, func(context.Context, toolapi.Approval) (bool, error) { return true, nil }, func(event runrecord.Event) error {
		if event.Agent == names[0] && event.Kind == "tool_succeeded" && event.Call != nil && event.Call.Name == "set_work" {
			close(g.claimed)
		}
		if event.Agent == names[1] && event.Kind == "tool_failed" && strings.Contains(event.Text, "team_work_conflict") {
			conflicts++
			close(g.conflict)
		}
		if event.Agent == names[0] && event.Kind == "team_observation" && strings.Contains(event.Text, "shared.txt") {
			close(g.changed)
		}
		if event.Kind == "tool_succeeded" && event.Call != nil && event.Call.Name == "write_file" {
			writes++
		}
		return nil
	})
	if err != nil || result.Status != runrecord.Answered || conflicts != 1 || writes != 2 || !g.sawOwner || !g.sawResult {
		t.Fatalf("result=%+v err=%v conflicts=%d writes=%d owner=%t facts=%t", result, err, conflicts, writes, g.sawOwner, g.sawResult)
	}
	body, err := os.ReadFile(filepath.Join(project, "shared.txt"))
	if err != nil || string(body) != "one implementation\n" {
		t.Fatal("duplicate implementation overwrote the file", err)
	}
	if len(service.CommandActivity()) != 0 {
		t.Fatal("closed members retained executors")
	}
}
