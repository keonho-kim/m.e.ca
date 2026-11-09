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
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
)

func (r Runner) start(ctx context.Context, input string, emit func(runrecord.Event) error, started time.Time) (*execution, func(), error) {
	var err error
	release := func() {}
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
	limits := r.Limits
	if limits == (Limits{}) {
		limits = DefaultLimits()
	}
	if err = limits.Check(); err != nil {
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
	id := "run_" + rand.Text()
	emit = newRunEmitter(ctx, r.Journal, r.Session, id, emit)
	if support, ok := r.Generator.(interface{ ImageInputSupported() bool }); ok {
		r.ImageInput = r.ImageInput && support.ImageInputSupported()
	}
	if scoped, ok := r.Generator.(interface {
		NewRunGenerator() (modelapi.Generator, func())
	}); ok {
		generator, releaseScope := scoped.NewRunGenerator()
		release = releaseScope
		r.Generator = generator
	}
	definitions := imageDefinitions(r.Tools.Definitions(), r.ImageInput)
	setSkillImageInput(r.Tools, r.ImageInput)
	for _, definition := range definitions {
		if definition.Name == "list_skills" {
			system += prompt.SkillTools
			break
		}
	}
	s := &execution{runner: r, limits: limits, system: system, emit: emit, definitions: definitions, statistics: runStatistics{started: started}}
	s.usage.begin(limits)
	if !r.Deadline.IsZero() {
		s.usage.deadline = r.Deadline
	}

	digest := sha256.Sum256([]byte(r.ProjectInstructions))
	s.checkpoint = runrecord.Checkpoint{ID: id, Objective: r.Redact(input), Status: runrecord.Running, Phase: "decision", InstructionsDigest: fmt.Sprintf("%x", digest)}
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
	releaseScope := release
	return s, func() { s.contextMemory.Close(); s.replyMemory.Close(); releaseScope() }, nil
}
