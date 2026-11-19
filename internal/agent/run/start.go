// Purpose: Validate one request and assemble its owned runtime resources before conversation work.
// Pattern: Execution composition boundary.
// Usage: Runner.Run receives the prepared execution and releases its scoped generator last.
// Related: internal/agent/run/run.go, internal/agent/run/emitter.go, internal/agent/run/conversation.go.
package run

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/presentation"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
)

func (r Runner) start(ctx context.Context, input string, emit func(runrecord.Event) error, started time.Time) (*execution, func(), error) {
	var err error
	if r.Generator == nil || r.Tools == nil || r.Journal == nil || r.Redact == nil || emit == nil {
		return nil, nil, faults.Errorf(diagcatalog.RunExecutionDependenciesAreRequired)
	}
	if r.PreviousRun != nil {
		if err := r.PreviousRun.Check(); err != nil {
			return nil, nil, err
		}
	}
	if err = r.Settings.Check(); err != nil {
		return nil, nil, err
	}
	if r.Inference != nil {
		if err := r.Inference.Check(); err != nil {
			return nil, nil, err
		}
	}
	limits, err := r.checkedLimits()
	if err != nil {
		return nil, nil, err
	}
	templates := prompt.Defaults()
	if r.Prompts != nil {
		templates = *r.Prompts
	}
	system, err := prompt.Work(templates.System, templates.Development, r.ProjectInstructions)
	if err != nil {
		return nil, nil, err
	}
	if len(input) > 64*1024*1024 {
		return nil, nil, faults.Errorf(diagcatalog.RunObjectiveExceedsMiB)
	}
	if support, ok := r.Generator.(interface{ ImageInputSupported() bool }); ok {
		r.ImageInput = r.ImageInput && support.ImageInputSupported()
	}
	if err := media.CheckInput(r.Images); err != nil {
		return nil, nil, err
	}
	if len(r.Images) > 0 && !r.ImageInput {
		return nil, nil, faults.New(diagcatalog.FailedPrecondition, diagcatalog.MediaImageInputUnavailable)
	}
	release := r.scopeGenerator()
	definitions := imageDefinitions(r.Tools.Definitions(), r.ImageInput)
	setSkillImageInput(r.Tools, r.ImageInput)
	guidance := make([]string, 1, 4)
	guidance[0] = system
	for _, definition := range definitions {
		switch definition.Name {
		case "list_skills":
			guidance = append(guidance, prompt.SkillTools)
		case toolapi.BrowserAction:
			guidance = append(guidance, prompt.BrowserAction)
		case toolapi.BrowserInspect:
			guidance = append(guidance, prompt.BrowserObservation)
		case toolapi.WebFetch:
			guidance = append(guidance, prompt.WebFetch)
		}
	}
	system = strings.Join(guidance, "")
	s := r.newExecution(ctx, limits, emit, started)
	s.system, s.definitions = system, definitions
	digest := sha256.Sum256([]byte(r.ProjectInstructions))
	s.checkpoint.Objective = r.Redact(input)
	s.checkpoint.InstructionsDigest = fmt.Sprintf("%x", digest)
	if r.PreviousRun != nil {
		for _, file := range r.PreviousRun.Work.Files {
			file.Changed = false
			s.checkpoint.Work.Files = append(s.checkpoint.Work.Files, file)
		}
	}
	s.contextMemory = processmemory.NewContextLease()
	s.replyMemory = processmemory.NewReplyLease()
	if err := s.reserveContext(0); err != nil {
		s.contextMemory.Close()
		s.replyMemory.Close()
		release()
		return nil, nil, err
	}
	return s, func() { s.contextMemory.Close(); s.replyMemory.Close(); release() }, nil
}

func (r Runner) newExecution(ctx context.Context, limits Limits, emit func(runrecord.Event) error, started time.Time) *execution {
	id := "run_" + rand.Text()
	s := &execution{runner: r, limits: limits, statistics: runStatistics{started: started},
		checkpoint: runrecord.Checkpoint{ID: id, Status: runrecord.Running, Phase: "decision"}}
	_, s.order = presentation.Scope(ctx)
	send := newRunEmitter(ctx, r.Journal, r.Session, id, emit)
	s.emit = func(event runrecord.Event) error {
		event = s.present(event)
		if event.Presentation == "" && (event.Kind == "retry" || event.Kind == "compaction_started" || event.Kind == "compaction_finished") {
			event.Presentation = s.order.Next()
		}
		return send(event)
	}
	s.usage.begin(limits)
	if !r.Deadline.IsZero() {
		s.usage.deadline = r.Deadline
	}
	return s
}

func (r *Runner) scopeGenerator() func() {
	if scoped, ok := r.Generator.(interface {
		NewRunGenerator() (modelapi.Generator, func())
	}); ok {
		generator, release := scoped.NewRunGenerator()
		r.Generator = generator
		return release
	}
	return func() {}
}
