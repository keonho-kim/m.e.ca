// Purpose: Verify silent tool calls receive recoverable feedback and public work plans reach teammates.
// Pattern: Scripted model integration test with real private stores and local tools.
// Usage: go test ./cmd/meca -run TestTeamCommunicatesBeforeLocalWork.
// Related: internal/commands/teamrequest/peer.go, internal/tools/team/admission.go.
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

type communicatingTeamModel struct {
	names         []string
	mu            sync.Mutex
	steps         map[string]int
	bAnnounced    chan struct{}
	bDiscussed    chan struct{}
	sawFeedback   bool
	sawPeer       bool
	sawDiscussion bool
}

func (g *communicatingTeamModel) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	id := g.names[0]
	if strings.Contains(request.Messages[0].Content, "Your peer name is "+g.names[1]+".") {
		id = g.names[1]
	}
	g.mu.Lock()
	step := g.steps[id]
	g.steps[id]++
	if id == g.names[0] && step == 1 {
		for _, message := range request.Messages {
			g.sawFeedback = g.sawFeedback || message.Role == "tool" && strings.Contains(message.Content, "team_contribution_required")
			g.sawPeer = g.sawPeer || message.Role == "user" && strings.Contains(message.Content, "Public conversation — "+g.names[1]) && strings.Contains(message.Content, "I will implement b.txt")
			g.sawDiscussion = g.sawDiscussion || message.Role == "user" && strings.Contains(message.Content, "The locked-account response is 423.")
		}
	}
	g.mu.Unlock()
	if id == g.names[0] && step == 0 {
		select {
		case <-g.bAnnounced:
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case <-g.bDiscussed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	var name, text string
	var args any
	switch {
	case id == g.names[0] && step == 0:
		name, args = "execute_command", map[string]string{"command": "printf unannounced > silent.txt"}
	case id == g.names[0] && step == 1:
		name, args = "set_work", map[string]any{"summary": "Implement a.txt alongside " + g.names[1], "paths": []string{"a.txt"}}
		text = g.names[1] + " is implementing b.txt; I will implement a.txt and keep b.txt unchanged."
	case id == g.names[1] && step == 0:
		name, args = "set_work", map[string]any{"summary": "Implement b.txt", "paths": []string{"b.txt"}}
		text = "I will implement b.txt. " + g.names[0] + " can handle a.txt independently."
	case id == g.names[0] && step == 2:
		name, args = "write_file", map[string]string{"path": "a.txt", "content": "A contribution\n"}
	case id == g.names[1] && step == 1:
		name, args = "discuss", map[string]any{"message": "The locked-account response is 423.", "mentions": []string{g.names[0]}}
	case id == g.names[1] && step == 2:
		name, args = "write_file", map[string]string{"path": "b.txt", "content": "B contribution\n"}
	case id == g.names[0] && step == 3 || id == g.names[1] && step == 3:
		name, args = "leave_team", map[string]string{"message": id + " completed its independent file."}
	default:
		return emit(modelapi.Event{Delta: "The contributions are complete.", Done: true})
	}
	body, _ := json.Marshal(args)
	return emit(modelapi.Event{Delta: text, Calls: []toolapi.Call{{ID: fmt.Sprintf("%s-%d", id, step), Name: name, Arguments: body}}, Done: true})
}

func TestTeamCommunicatesBeforeLocalWork(t *testing.T) {
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
	var names []string
	for range 2 {
		member, err := store.CreateTeamAgent(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, member.ID)
	}
	model := &communicatingTeamModel{steps: map[string]int{}, names: names, bAnnounced: make(chan struct{}), bDiscussed: make(chan struct{})}
	spoke := map[string]bool{}
	feedback := 0
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := service.Run(ctx, agentrequest.Options{Runtime: agentrequest.Runtime{Config: configuration, Session: store.Session(), Selection: connection.Selection{Model: "test-model"}, Vault: vault}, Text: "Create a.txt and b.txt together; coordinate separate contributions.", Generator: model}, func(context.Context, toolapi.Approval) (bool, error) { return true, nil }, func(event runrecord.Event) error {
		if event.Kind == "team_progress" {
			if event.Agent == names[1] && !spoke[event.Agent] {
				close(model.bAnnounced)
			}
			spoke[event.Agent] = true
		}
		if event.Kind == "team_discussion" && event.Agent == names[1] {
			close(model.bDiscussed)
		}
		if event.Kind == "tool_started" && event.Call != nil && event.Call.Name != "leave_team" && !spoke[event.Agent] {
			t.Errorf("%s started %s without a public contribution", event.Agent, event.Call.Name)
		}
		if event.Kind == "tool_failed" {
			var rejected struct {
				Executed bool             `json:"executed"`
				Failure  *toolapi.Failure `json:"failure"`
			}
			if json.Unmarshal(event.Output, &rejected) == nil && rejected.Failure != nil && rejected.Failure.Code == "team_contribution_required" && !rejected.Executed {
				feedback++
			}
		}
		return nil
	})
	if err != nil || result.Status != runrecord.Answered {
		t.Fatalf("team result=%+v error=%v", result, err)
	}
	if _, err := os.Stat(filepath.Join(project, "silent.txt")); !os.IsNotExist(err) {
		t.Fatalf("unannounced command was allowed to execute: %v", err)
	}
	if feedback != 1 || !model.sawFeedback || !model.sawPeer || !model.sawDiscussion {
		t.Fatalf("feedback=%d received=%t peer=%t discussion=%t", feedback, model.sawFeedback, model.sawPeer, model.sawDiscussion)
	}
	for _, file := range []string{"a.txt", "b.txt"} {
		if _, err := os.Stat(filepath.Join(project, file)); err != nil {
			t.Fatal(err)
		}
	}
}
