// Purpose: Own the shared settings, access gate and approval boundary of one team request.
// Pattern: Request-scoped application composition.
// Usage: Service.Run snapshots prompts and dispatches independent peer lifetimes through team.Run.
// Related: internal/commands/teamrequest/service.go, internal/commands/teamrequest/peer.go.
package teamrequest

import (
	"context"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/agent/team"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/tools/team/approval"
	"github.com/keonho-kim/meca/internal/tools/team/workspace"
	"sync"
)

type request struct {
	service             *Service
	options             agentrequest.Options
	templates           prompt.Templates
	access              *workspace.Access
	approver            run.Approver
	approval            sync.Mutex
	refusals            approval.Ledger
	images              chan struct{}
	instructions        sync.Once
	projectInstructions string
	instructionsErr     error
}

func (s *Service) Run(ctx context.Context, options agentrequest.Options, approve run.Approver, emit func(runrecord.Event) error) (team.Result, error) {
	if s.Open == nil || s.NewTools == nil || s.LoadPrompts == nil || s.Repository == nil || s.Record == nil || approve == nil || emit == nil {
		return team.Result{}, fmt.Errorf("team I/O factories and callbacks are required")
	}
	templates, err := s.LoadPrompts()
	if err != nil {
		return team.Result{}, err
	}
	current := &request{service: s, options: options, templates: templates, access: workspace.New(), images: make(chan struct{}, 1), approver: approve}
	return team.Run(ctx, team.Options{Store: s.Repository, Execute: current.execute, Emit: emit, Record: s.Record, PollSteering: options.PollSteering, SteeringReady: options.SteeringReady, Redact: options.Vault.Redact, MaxModelCalls: options.Config.Loop.MaxModelCalls, MaxSeconds: options.Config.Loop.MaxSeconds, ProjectRoot: options.Session.ProjectRoot}, options.Text)
}
