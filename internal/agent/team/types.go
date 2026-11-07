// Purpose: Define persistent identities and the narrow collaboration/runtime contracts.
// Pattern: Consumer-owned application contracts.
// Usage: SQLite and request composition supply Repository and Execute to Room.
// Related: internal/agent/team/room.go, internal/commands/teamrequest/request.go.
package team

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/contribution"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/agent/work"
)

const MaxActive = 8

type MemberStatus struct {
	ID      string              `json:"member_name"`
	Summary string              `json:"summary,omitempty"`
	Active  bool                `json:"active"`
	Work    *contribution.Scope `json:"work,omitempty"`
}
type MemberPage struct {
	Members []MemberStatus `json:"members"`
	Next    string         `json:"next,omitempty"`
}

type Repository interface {
	TeamMembers(context.Context, string, int) ([]identity.Member, error)
	TeamMember(context.Context, string) (identity.Member, error)
	CreateTeamAgent(context.Context) (identity.Member, error)
	DiscardTeamAgent(context.Context, identity.Member) error
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
	SteeringReady <-chan struct{}
	Redact        func(string) string
	MaxModelCalls int
	MaxSeconds    int
	ProjectRoot   string
}
type Result struct {
	End         work.End
	Text, Agent string
	Status      runrecord.Status
	Started     time.Time
}
