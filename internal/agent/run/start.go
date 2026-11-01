// Purpose: Validate one request and assemble its owned runtime resources before conversation work.
// Pattern: Execution composition boundary.
// Usage: Runner.Run receives the prepared execution and releases its scoped generator last.
// Related: internal/agent/run/run.go, internal/agent/run/runner.go, internal/agent/run/conversation.go.
package run

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (r Runner) start(ctx context.Context, input string, emit func(runrecord.Event) error, started time.Time) (*execution, func(), error) {
	var err error
	release := func() {}
	if r.Generator == nil || r.Tools == nil || r.Journal == nil || r.Redact == nil || emit == nil {
		return nil, nil, fmt.Errorf("execution dependencies are required")
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
	var system string
	if r.fork != nil {
		system = r.fork.system
	} else {
		system, err = prompt.Work(templates.System, templates.Development, r.ProjectInstructions)
		if err != nil {
			return nil, nil, err
		}
	}
	if len(input) > 64*1024*1024 {
		return nil, nil, fmt.Errorf("objective exceeds 64 MiB")
	}
	originalEmit := emit
	emit = func(event runrecord.Event) error {
		if err := originalEmit(event); err != nil {
			if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
				return ctx.Err()
			}
			return errors.Join(errRuntimeRecord, err)
		}
		return nil
	}
	if support, ok := r.Generator.(interface{ ImageInputSupported() bool }); ok {
		r.ImageInput = r.ImageInput && support.ImageInputSupported()
	}
	reviewGenerator := r.Generator
	if scoped, ok := r.Generator.(interface {
		NewRunGenerator() (modelapi.Generator, func())
	}); ok {
		generator, releaseScope := scoped.NewRunGenerator()
		release = releaseScope
		r.Generator = generator
	}
	var definitions []toolapi.Definition
	if r.fork != nil {
		definitions = r.fork.definitions
	} else {
		definitions = imageDefinitions(r.Tools.Definitions(), r.ImageInput)
	}
	s := &execution{runner: r, limits: limits, system: system, emit: emit, definitions: definitions, statistics: runStatistics{started: started}}
	if r.LAT {
		s.review.generator = reviewGenerator
	}
	s.usage = &requestBudget{}
	if r.fork != nil {
		s.usage = r.fork.budget
		s.review.active = true
	} else {
		s.usage.begin(limits)
	}
	if r.LAT {
		owner, stop := context.WithDeadline(ctx, s.usage.snapshot().deadline)
		s.review.memory = &reviewMemory{ctx: owner, stop: stop, dirty: true}
		s.review.memory.controller.Repository, s.review.memory.controller.Session = r.ReviewMemory, r.Session
		s.review.memory.controller.Redact = r.Redact
		s.review.memory.controller.RedactionVersion = r.RedactionVersion
	}

	digest := sha256.Sum256([]byte(r.ProjectInstructions))
	id := "run_" + rand.Text()
	s.checkpoint = runrecord.Checkpoint{ID: id, Objective: r.Redact(input), Status: runrecord.Running, Phase: "decision", InstructionsDigest: fmt.Sprintf("%x", digest)}
	if r.PreviousRun != nil {
		for _, file := range r.PreviousRun.Work.Files {
			file.Changed = false
			s.checkpoint.Work.Files = append(s.checkpoint.Work.Files, file)
		}
	}
	return s, release, nil
}
