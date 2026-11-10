// Purpose: Keep command status bound to execution resources while other clients select another session.
// Pattern: Application ownership contract test with a synchronized source substitute.
// Usage: go test -race ./internal/commands/server -run TestCommandsUse.
// Related: internal/commands/server/command_activity.go, internal/commands/server/task_registry.go.
package server

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"testing"
)

type commandTools struct{ agentrequest.Tools }

func (commandTools) CommandActivity() []toolapi.CommandRun {
	return []toolapi.CommandRun{{ID: "cmd_owned", CallID: "call_owned", Command: "inspect", Status: "running", ElapsedMillis: 42}}
}

func TestCommandsUseExecutionSessionAfterSelectionChanges(t *testing.T) {
	s := detailFixture(&detailHistory{}, nil)
	defer s.Close()
	owner := s.conversation
	owner.Runtime.Tools = commandTools{}
	s.active = map[string]*request{"task-owned": {task: Task{ID: "task-owned", Session: owner.Runtime.Session.ID, Status: "running"}, conversation: owner}}
	s.jobs = map[string]*Conversation{owner.Runtime.Session.ID: owner}
	s.conversation = &Conversation{Close: func() error { return nil }}
	values, err := s.Commands(t.Context(), "task-owned")
	if err != nil || len(values) != 1 || values[0].Task != "task-owned" || values[0].Run.ID != "cmd_owned" {
		t.Fatal(values, err)
	}
}
