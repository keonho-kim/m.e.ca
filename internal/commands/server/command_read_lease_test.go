// Purpose: Keep a command status source alive until its reader finishes across task retirement and selection changes.
// Pattern: Controlled lifecycle regression test.
// Usage: go test -race ./internal/commands/server -run TestCommandReaderPins.
// Related: internal/commands/server/command_activity.go, internal/commands/server/lifecycle.go.
package server

import (
	"context"
	"testing"
	"testing/synctest"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/config"
)

type heldCommandSource struct {
	agentrequest.Tools
	entered, release chan struct{}
}

func (s heldCommandSource) CommandActivity() []toolapi.CommandRun {
	close(s.entered)
	<-s.release
	return []toolapi.CommandRun{{ID: "owned-command", Status: "running"}}
}

func TestCommandReaderPinsRetiredExecutionResources(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := detailFixture(&detailHistory{}, nil)
		defer s.Close()
		owner := s.conversation
		source := heldCommandSource{entered: make(chan struct{}), release: make(chan struct{})}
		owner.Runtime.Tools = source
		s.active = map[string]*request{"owned-task": {task: Task{ID: "owned-task", Session: owner.Runtime.Session.ID, Status: "running"}, conversation: owner}}
		readDone := make(chan struct{})
		go func() { _, _ = s.Commands(t.Context(), "owned-task"); close(readDone) }()
		<-source.entered
		s.mu.Lock()
		delete(s.active, "owned-task")
		s.mu.Unlock()
		closed := false
		owner.Close = func() error { closed = true; return nil }
		s.options.Open = func(context.Context, string, bool, config.Config) (*Conversation, error) {
			return &Conversation{Close: func() error { return nil }}, nil
		}
		selected := make(chan error, 1)
		go func() { _, err := s.Select(t.Context(), "", true); selected <- err }()
		synctest.Wait()
		if closed {
			t.Error("selection closed resources while command metadata was borrowed")
		}
		close(source.release)
		<-readDone
		if err := <-selected; err != nil {
			t.Fatal(err)
		}
		if !closed {
			t.Fatal("retired resources remained open after the reader finished")
		}
	})
}
