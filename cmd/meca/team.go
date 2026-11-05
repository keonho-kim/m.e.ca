// Purpose: Wire private team stores, leases and local executors at the executable boundary.
// Pattern: Composition root.
// Usage: Interactive and restored headless requests use this service when team mode is enabled.
// Related: internal/commands/teamrequest/request.go, cmd/meca/conversation.go.
package main

import (
	"context"
	"errors"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/teamrequest"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/config/prompts"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local"
	teamtools "github.com/keonho-kim/meca/internal/tools/team"
)

func teamService(root *sqlite.Store, home, state string, vault *credentials.Vault) *teamrequest.Service {
	return &teamrequest.Service{
		Repository: root,
		Open: func(ctx context.Context, id string) (teamrequest.Child, error) {
			store, err := root.OpenTeamAgent(ctx, id)
			if err != nil {
				return teamrequest.Child{}, err
			}
			lease, err := store.LeaseSession()
			if err != nil {
				return teamrequest.Child{}, errors.Join(err, store.Close())
			}
			return teamrequest.Child{Store: store, Session: store.Session(), Close: func() error { return errors.Join(store.Close(), lease.Close()) }}, nil
		},
		NewTools: func() (teamtools.Local, error) {
			executor, err := local.NewExecutor(root.Session().ProjectRoot, home, state, vault.Redact)
			if err == nil {
				executor.SetRedactionVersion(vault.RedactionVersion)
			}
			return executor, err
		},
		LoadPrompts: func() (prompt.Templates, error) { return prompts.LoadProfile(state, prompt.Team) },
		Record: func(ctx context.Context, event runrecord.Event) error {
			return root.RecordEvent(ctx, root.Session().ID, event)
		},
	}
}
