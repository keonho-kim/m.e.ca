// Purpose: Compose real public and private team journals for server integration tests.
// Pattern: Test composition root mirroring the executable's team factories.
// Usage: newFixture attaches this service; tests explicitly enable team execution.
// Related: cmd/meca/team.go, internal/serverapi/fixture_test.go.
package serverapi

import (
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/teamrequest"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local"
	teamtools "github.com/keonho-kim/meca/internal/tools/team"
)

func fixtureTeam(root *sqlite.Store, home, state string, vault *credentials.Vault) *teamrequest.Service {
	return &teamrequest.Service{Repository: root, Record: func(ctx context.Context, e record.Event) error { return root.RecordEvent(ctx, root.Session().ID, e) },
		LoadPrompts: func() (prompt.Templates, error) { return prompt.Defaults(), nil },
		Open: func(ctx context.Context, id string) (teamrequest.Child, error) {
			child, err := root.OpenTeamAgent(ctx, id)
			if err != nil {
				return teamrequest.Child{}, err
			}
			lease, err := child.LeaseSession()
			if err != nil {
				return teamrequest.Child{}, errors.Join(err, child.Close())
			}
			return teamrequest.Child{Store: child, Session: child.Session(), Close: func() error { return errors.Join(child.Close(), lease.Close()) }}, nil
		},
		NewTools: func(options teamrequest.LocalOptions) (teamtools.Local, error) {
			return local.NewExecutor(root.Session().ProjectRoot, home, state, vault.Redact, local.Options{Skills: options.Skills, Images: options.Images, RedactRange: vault.RedactRange, RedactionVersion: vault.RedactionVersion})
		},
	}
}
