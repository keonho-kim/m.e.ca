// Purpose: Stop using an old model when a replacement's durable selection cannot be established.
// Pattern: Persistence failure and ownership regression test.
// Usage: go test ./internal/commands/server -run TestUncertainModelSave.
// Related: internal/commands/server/settings.go, internal/commands/modelchange/commit.go.
package server

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/conversation"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
)

type unreadableSelection struct{}

func (unreadableSelection) SaveSelection(context.Context, string, connection.Selection) error {
	return errors.New("commit acknowledgement lost")
}
func (unreadableSelection) Selection(context.Context, string) (connection.Selection, error) {
	return connection.Selection{}, errors.New("database unavailable")
}

func TestUncertainModelSaveInvalidatesLiveConnection(t *testing.T) {
	s := detailFixture(&detailHistory{}, &credentials.Vault{})
	defer s.Close()
	var previousClosed, candidateClosed atomic.Int32
	s.conversation.Model = conversation.NewModel(nil, func() { previousClosed.Add(1) })
	s.conversation.Runtime.Selection = connection.Selection{Model: "old"}
	s.conversation.Models = unreadableSelection{}
	s.options.PrepareModel = func(context.Context, connection.Selection, string, setup.Defaults) (PreparedModel, error) {
		return PreparedModel{Selection: connection.Selection{Model: "new"}, Close: func() { candidateClosed.Add(1) }}, nil
	}
	state, err := s.SetModel(t.Context(), ModelSettings{Provider: "ollama", Name: "new"})
	if err == nil || state.Selection.Model != "" || s.conversation.Model != nil {
		t.Fatalf("uncertain write left an active model: %+v %v", state.Selection, err)
	}
	if previousClosed.Load() != 1 || candidateClosed.Load() != 1 {
		t.Fatal("connection owner not released", previousClosed.Load(), candidateClosed.Load())
	}
}
