// Purpose: Declare team request I/O factories and the live command-source registry.
// Pattern: Consumer-owned composition contracts.
// Usage: cmd/meca supplies concrete storage and local executors to Service.
// Related: internal/commands/teamrequest/request.go, internal/commands/teamrequest/peer.go.
package teamrequest

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	agentsession "github.com/keonho-kim/meca/internal/agent/session"
	"github.com/keonho-kim/meca/internal/agent/team"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	teamtools "github.com/keonho-kim/meca/internal/tools/team"
	"sync"
)

type Store interface {
	agentsession.Store
	ContextCursor(context.Context, string) (int64, error)
}
type Child struct {
	Store   Store
	Session sessionstate.Session
	Close   func() error
}
type Service struct {
	Repository  team.Repository
	Open        func(context.Context, string) (Child, error)
	NewTools    func() (teamtools.Local, error)
	LoadPrompts func() (prompt.Templates, error)
	Record      func(context.Context, runrecord.Event) error
	mu          sync.Mutex
	active      map[string]commandSource
}
