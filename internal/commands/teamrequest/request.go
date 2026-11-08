// Purpose: Own the shared settings, access gate and approval boundary of one team request.
// Pattern: Request-scoped application composition.
// Usage: Service.Run snapshots prompts and dispatches independent peer lifetimes through team.Run.
// Related: internal/commands/teamrequest/service.go, internal/commands/teamrequest/peer.go.
package teamrequest

import (
	"context"
	"sync"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/agent/team"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/skills/instruction"
	"github.com/keonho-kim/meca/internal/tools/local/process"
	"github.com/keonho-kim/meca/internal/tools/team/approval"
	"github.com/keonho-kim/meca/internal/tools/team/workspace"
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
		return team.Result{}, faults.Errorf(diagcatalog.TeamrequestTeamIOFactoriesAndCallbacksAre)
	}
	templates, err := s.LoadPrompts()
	if err != nil {
		return team.Result{}, err
	}
	if source, ok := options.Tools.(interface{ Skills() *skills.Reader }); ok && source.Skills() != nil {
		for _, name := range instruction.Mentions(options.Text) {
			doc, err := source.Skills().Load(ctx, instruction.Selector{ID: name}, true)
			if err != nil {
				return team.Result{}, err
			}
			options.SkillReads = append(options.SkillReads, doc)
		}
	}
	current := &request{service: s, options: options, templates: templates, access: workspace.New(), images: make(chan struct{}, 1), approver: approve}
	ctx = process.WithProjectionBudget(ctx)
	return team.Run(ctx, team.Options{Store: s.Repository, Execute: current.execute, Emit: emit, Record: s.Record, PollSteering: options.PollSteering, SteeringReady: options.SteeringReady, Redact: options.Vault.Redact, MaxModelCalls: options.Config.Loop.MaxModelCalls, MaxSeconds: options.Config.Loop.MaxSeconds, ProjectRoot: options.Session.ProjectRoot}, options.Text)
}
