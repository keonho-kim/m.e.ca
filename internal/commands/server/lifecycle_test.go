// Purpose: Keep state queries responsive during setup and publish irreversible session handoffs on failure.
// Pattern: Controlled I/O lifecycle regression tests.
// Usage: go test -race ./internal/commands/server -run 'TestStateDuring|TestFailedSession'.
// Related: internal/commands/server/service.go, internal/commands/server/settings_test.go.
package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
)

func TestStateDuringModelPreparation(t *testing.T) {
	s := detailFixture(&detailHistory{}, &credentials.Vault{})
	defer s.Close()
	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	s.options.PrepareModel = func(context.Context, connection.Selection, string, setup.Defaults) (PreparedModel, error) {
		close(entered)
		<-release
		return PreparedModel{}, errors.New("controlled setup failure")
	}
	go func() {
		defer close(finished)
		_, _ = s.SetModel(t.Context(), ModelSettings{Session: s.State().Session.ID, Provider: "ollama", Name: "fixture"})
	}()
	<-entered
	queried := make(chan struct{})
	go func() { _ = s.State(); close(queried) }()
	deadline, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	select {
	case <-queried:
	case <-deadline.Done():
		t.Error("state query waited for provider I/O")
	}
	close(release)
	<-finished
	<-queried
}

func TestFailedSessionHandoffPublishesActualState(t *testing.T) {
	s := detailFixture(&detailHistory{}, &credentials.Vault{})
	defer s.Close()
	oldErr, nextErr := errors.New("old close failed"), errors.New("new close failed")
	s.conversation.Close = func() error { return oldErr }
	s.publish(Event{Record: eventKind("session_opened")})
	previous := s.State()
	s.options.Open = func(context.Context, string, bool, config.Config) (*Conversation, error) {
		return &Conversation{Close: func() error { return nextErr }}, nil
	}
	_, err := s.Select(t.Context(), "", true)
	if !errors.Is(err, oldErr) || !errors.Is(err, nextErr) {
		t.Error("handoff did not preserve every cleanup error", err)
	}
	actual := s.State()
	if actual.Session.ID != "" || actual.Cursor == previous.Cursor {
		t.Error("failed handoff left observers with stale selected state")
	}
	batch, err := s.Replay(previous.Cursor)
	if err != nil || batch.Snapshot != nil || len(batch.Events) != 1 || batch.Events[0].Record.Kind != "session_closed" || batch.Events[0].Session != previous.Session.ID {
		t.Fatal("previous cursor lost the ordered session closure", batch, err)
	}
}
