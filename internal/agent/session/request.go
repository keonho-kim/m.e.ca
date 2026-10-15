// Purpose: Prepare restored context and execute one submitted task or approved Plan.
// Pattern: Use Case.
// Usage: The terminal worker calls Execute with a snapshot of the selected runtime.
// Related: internal/agent/run/runner.go, internal/storage/recovery/recovery.go.
package session

import (
	"context"
	"errors"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	"github.com/keonho-kim/v-code/internal/agent/monitor"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/storage/recovery"
)

type Store interface {
	recovery.Store
	run.Journal
	compact.Repository
	MonitorHistory(context.Context, string) ([]monitor.Record, error)
}
type Request struct {
	Runner              run.Runner
	Store               Store
	LoadPrompts         func() (prompt.Templates, error)
	Compaction          compact.Settings
	ProjectInstructions func(context.Context) (string, error)
	Text                string
	Planning            bool
	Previous, Approved  *planning.Plan
}

func Execute(ctx context.Context, request Request, approve run.Approver, emit func(run.Event) error) (*planning.Plan, error) {
	runner := request.Runner
	var err error
	runner.PreviousRun, err = request.Store.LatestRun(ctx, runner.Session)
	if err != nil {
		return nil, err
	}
	runner.MonitorSeed, err = request.Store.MonitorHistory(ctx, runner.Session)
	if err != nil {
		return nil, err
	}
	templates, err := request.LoadPrompts()
	if err != nil {
		return nil, err
	}
	for _, text := range templates.Files() {
		*text = runner.Redact(*text)
	}
	runner.Prompts = &templates
	runner.Memory = &compact.Controller{Inference: runner.Inference, Repository: request.Store, Session: runner.Session, Model: runner.Model, Prompt: templates.Compaction, Settings: request.Compaction, Redact: runner.Redact}
	runner.ProjectInstructions, err = request.ProjectInstructions(ctx)
	if err != nil {
		return nil, err
	}
	var updated *planning.Plan
	if request.Approved != nil {
		p, executionErr := runner.ExecutePlan(ctx, *request.Approved, request.Store, approve, emit)
		updated, err = &p, executionErr
	} else if request.Planning {
		updated, err = runner.DraftPlan(ctx, request.Text, request.Previous, request.Store, approve, emit)
	} else {
		_, err = runner.Run(ctx, request.Text, false, approve, emit)
	}
	if err != nil {
		err = errors.Join(err, recovery.Restore(context.WithoutCancel(ctx), request.Store, runner.Session))
	}
	return updated, err
}
