// Purpose: Compose project-scoped startup selection and open only the chosen conversation.
// Pattern: Composition Root.
// Usage: start calls selectConversation before wiring a session's tools and model.
// Related: cmd/v/main.go, internal/tui/startup/resume.go, internal/storage/sqlite/catalog.go.
package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/commands/startup"
	sessionstate "github.com/keonho-kim/v-code/internal/storage/session"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	tuistartup "github.com/keonho-kim/v-code/internal/tui/startup"
)

func selectConversation(ctx context.Context, screen tcell.Screen, sessions *sqlite.Sessions, mode startup.Mode, cwd string) (*sqlite.Store, bool, error) {
	all := mode.Resume
	if all {
		var err error
		mode, err = startup.Parse(nil, cwd)
		if err != nil {
			return nil, false, err
		}
	}
	load := sessions.Conversations
	if !all {
		load = func(ctx context.Context, offset, limit int) ([]sessionstate.Conversation, error) {
			return sessions.ProjectConversations(ctx, mode.Project, offset, limit)
		}
	}
	selected, ok, err := tuistartup.Select(ctx, screen, load, sessionstate.Session{ProjectRoot: mode.Project, DisplayRoot: mode.Display})
	if err != nil || !ok {
		return nil, false, err
	}
	if selected.ID == "" {
		store, err := sessions.Create(ctx, selected.ProjectRoot, selected.DisplayRoot)
		return store, err == nil, err
	}
	store, err := sessions.Open(ctx, selected.ProjectRoot, selected.ID)
	if err != nil {
		return nil, false, err
	}
	if store.Session() != selected {
		return nil, false, errors.Join(fmt.Errorf("selected session metadata changed; select it again"), store.Close())
	}
	return store, true, nil
}
