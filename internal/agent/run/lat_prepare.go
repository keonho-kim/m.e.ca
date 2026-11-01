// Purpose: Prepare and refine one implementation plan before normal execution.
// Pattern: Optional request boundary with advisory failure handling.
// Usage: Runner calls preparation once after saving the user's original request.
// Related: internal/agent/run/lat_fork.go, internal/agent/lat/plan.go.
package run

import (
	"context"
	"errors"

	"github.com/keonho-kim/meca/internal/agent/inference"
	"github.com/keonho-kim/meca/internal/agent/lat"
	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (s *execution) prepareLAT(ctx context.Context) error {
	if !s.runner.LAT {
		return nil
	}
	s.review.memory.evidence = &latmemory.Evidence{}
	defer func() { s.review.memory.evidence.Clear(); s.review.memory.evidence = nil }()
	s.review.active = true
	defer func() { s.review.active = false }()
	for {
		plan, err := s.refinedLATPlan(ctx)
		if ctx.Err() != nil {
			return errors.Join(ctx.Err(), err)
		}
		if errors.Is(err, errRuntimeRecord) {
			return err
		}
		if s.runner.PollSteering != nil {
			s.review.steering = append(s.review.steering, s.runner.PollSteering()...)
		}
		if len(s.review.steering) > 0 {
			s.review.memory.evidence.Clear()
			messages := s.review.steering
			s.review.steering = nil
			if err := s.applyInstructions(ctx, messages); err != nil {
				return err
			}
			if err := s.reviewEvent(ctx, "lat_discarded", ""); err != nil {
				return err
			}
			continue // Only new human input can restart preparation.
		}
		if err != nil {
			return s.reviewEvent(ctx, "lat_unavailable", s.runner.Redact(err.Error()))
		}
		plan = s.runner.Redact(plan)
		if err := s.record(ctx, runrecord.Event{Kind: "lat_plan", Text: plan}); err != nil {
			return err
		}
		s.review.plan = plan
		s.messages = append(s.messages, lat.Plan(plan))
		return s.emit(runrecord.Event{Kind: "lat_prepared"})
	}
}

func (s *execution) refinedLATPlan(ctx context.Context) (string, error) {
	templates := s.latTemplates()
	plan, err := s.runLATPhase(ctx, "plan", templates.LATPlan, inference.Agent, lat.Input{})
	if err != nil {
		return "", err
	}
	critique, err := s.runLATPhase(ctx, "critique", templates.LATCritique, inference.LAT, lat.Input{Plan: plan})
	if err != nil {
		return "", err
	}
	return s.runLATPhase(ctx, "refine", templates.LATRefine, inference.Agent, lat.Input{Plan: plan, Critique: critique})
}
