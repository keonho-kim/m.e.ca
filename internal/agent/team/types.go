// Purpose: Define persistent identities and the narrow collaboration/runtime contracts.
// Pattern: Consumer-owned application contracts.
// Usage: SQLite and request composition supply Repository and Execute to Room.
// Related: internal/agent/team/room.go, internal/commands/teamrequest/request.go.
package team

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
)

const MaxActive = 8

type Friend struct {
	ID      string `json:"id"`
	Summary string `json:"summary,omitempty"`
	Active  bool   `json:"active"`
}
type FriendPage struct {
	Friends []Friend `json:"friends"`
	Next    string   `json:"next,omitempty"`
}

type Repository interface {
	TeamMembers(context.Context, string, int) ([]identity.Member, error)
	TeamMember(context.Context, string) (identity.Member, error)
	CreateTeamAgent(context.Context) (identity.Member, error)
	SetTeamSummary(context.Context, string, string) error
	channel.Repository
}

type Execute func(context.Context, identity.Member, string, *Peer) (run.Result, error)
type Options struct {
	Store         Repository
	Execute       Execute
	Emit          func(runrecord.Event) error
	Record        func(context.Context, runrecord.Event) error
	PollSteering  func() []string
	Redact        func(string) string
	MaxModelCalls int
	MaxSeconds    int
}
type Result struct {
	Text, Agent string
	Status      runrecord.Status
	Started     time.Time
}
