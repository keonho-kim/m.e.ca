// Purpose: Prepare restored context and execute one submitted user request.
// Pattern: Use Case.
// Usage: TUI and headless commands call Execute with a selected runtime snapshot.
// Related: internal/agent/run/runner.go, internal/storage/recovery/recovery.go.
package session

import (
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/agent/compact"
	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/storage/recovery"
)

type Store interface {
	recovery.Store
	run.Journal
	compact.Repository
	latmemory.Repository
}
type Request struct {
	Runner              run.Runner
	Store               Store
	LoadPrompts         func() (prompt.Templates, error)
	LoadLAT             func() (prompt.Templates, error)
	ContextWindow       int
	Compaction          compact.Settings
	ProjectInstructions func(context.Context) (string, error)
	Text                string
}

func Execute(ctx context.Context, request Request, approve run.Approver, emit func(runrecord.Event) error) (run.Result, error) {
	runner := request.Runner
	var err error
	runner.PreviousRun, err = request.Store.LatestRun(ctx, runner.Session)
	if err != nil {
		return run.Result{}, err
	}
	templates, err := request.LoadPrompts()
	if err != nil {
		return run.Result{}, err
	}
	if runner.LAT && request.LoadLAT != nil {
		latTemplates, loadErr := request.LoadLAT()
		err = loadErr
		if err != nil {
			return run.Result{}, err
		}
		templates.LAT, templates.LATCompaction = latTemplates.LAT, latTemplates.LATCompaction
	}
	for _, text := range templates.Files() {
		*text = runner.Redact(*text)
	}
	runner.Prompts = &templates
	if runner.LAT {
		runner.ReviewMemory = request.Store
	}
	runner.Memory = &compact.Controller{ContextWindow: request.ContextWindow, Inference: runner.Inference, Repository: request.Store, Session: runner.Session, Model: runner.Model, Prompt: templates.Compaction, Settings: request.Compaction, Redact: runner.Redact}
	runner.ProjectInstructions, err = request.ProjectInstructions(ctx)
	if err != nil {
		return run.Result{}, err
	}
	result, err := runner.Run(ctx, request.Text, approve, emit)
	if err != nil {
		err = errors.Join(err, recovery.Restore(context.WithoutCancel(ctx), request.Store, runner.Session))
	}
	return result, err
}
