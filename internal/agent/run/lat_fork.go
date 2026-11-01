// Purpose: Reassemble the native main loop as an isolated read-only LAT session.
// Pattern: Request-owned fork with explicit resource and budget ownership.
// Usage: LAT preparation and final review open one fork per phase.
// Related: internal/agent/lat/transcript/journal.go, internal/tools/local/read_only.go.
package run

import (
	"context"
	"errors"
	"fmt"

	"github.com/keonho-kim/meca/internal/agent/inference"
	"github.com/keonho-kim/meca/internal/agent/lat"
	"github.com/keonho-kim/meca/internal/agent/lat/transcript"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type forkConfig struct {
	role        inference.Role
	maxOutput   int
	budget      *requestBudget
	system      string
	definitions []toolapi.Definition
}

func (s *execution) inferenceRole() inference.Role {
	if s.runner.fork != nil {
		return s.runner.fork.role
	}
	return inference.Agent
}

func (s *execution) latTemplates() prompt.Templates {
	if s.runner.Prompts != nil {
		return *s.runner.Prompts
	}
	return prompt.Defaults()
}

func (s *execution) runLATPhase(ctx context.Context, phase, instructions string, role inference.Role, input lat.Input) (text string, err error) {
	var scope *runrecord.LATRun
	defer func() {
		if scope == nil {
			return
		}
		event := runrecord.Event{Kind: "lat_phase_finished", Internal: true, LAT: scope}
		if err != nil {
			event.Text = s.runner.Redact(err.Error())
		}
		err = errors.Join(err, s.emit(event))
	}()
	if phase != "review" {
		if err := s.emit(runrecord.Event{Kind: "lat_preparing", Text: phase}); err != nil {
			return "", err
		}
	}
	if err := s.checkTimeBudget(ctx); err != nil {
		return "", err
	}
	if s.usage.snapshot().ModelCalls >= s.limits.MaxModelCalls {
		return "", errModelBudget
	}
	factory, ok := s.runner.Tools.(interface {
		ReadOnly() (toolapi.Executor, func() error, error)
	})
	if !ok {
		return "", fmt.Errorf("read-only LAT tools are unavailable")
	}
	tools, closeTools, err := factory.ReadOnly()
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, closeTools()) }()
	journal, err := transcript.New(s.runner.Journal, s.runner.Session, s.checkpoint.ID, phase)
	if err != nil {
		return "", err
	}
	defer journal.Close()
	identity := journal.Scope()
	scope = &identity
	if err := s.emit(runrecord.Event{Kind: "lat_phase_started", Internal: true, LAT: scope}); err != nil {
		return "", err
	}

	packet, definitions, err := s.prepareLATPacket(ctx, instructions, role, input, tools)
	if err != nil {
		return "", err
	}

	child := s.latRunner(instructions, role, packet, definitions, tools, journal)

	owner, cancel := context.WithDeadline(ctx, s.usage.snapshot().deadline)
	defer cancel()
	child.PollSteering = func() []string {
		if s.runner.PollSteering != nil {
			if messages := s.runner.PollSteering(); len(messages) > 0 {
				s.review.steering = append(s.review.steering, messages...)
				cancel()
			}
		}
		return nil
	}
	result, err := child.Run(owner, packet.Content, func(context.Context, toolapi.Approval) (bool, error) {
		return false, fmt.Errorf("LAT cannot authorize operations")
	}, func(event runrecord.Event) error {
		event.Internal, event.LAT = true, scope
		return s.forwardLATEvent(event)
	})
	if len(s.review.steering) > 0 {
		return "", errors.Join(errReviewSuperseded, err)
	}
	if err != nil {
		return "", err
	}
	if result.Status != runrecord.Answered {
		return "", fmt.Errorf("LAT %s stopped before a final report: %s", phase, result.Text)
	}
	return result.Text, nil
}
