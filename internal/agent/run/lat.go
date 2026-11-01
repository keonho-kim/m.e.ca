// Purpose: Run an optional read-only native session before finishing a request.
// Pattern: Request-owned optional phase with advisory feedback.
// Usage: The normal loop calls reviewWork after steering and active commands settle.
// Related: internal/agent/lat/packet.go, internal/agent/run/usage.go, internal/agent/run/finish.go.
package run

import (
	"context"
	"errors"
	"strings"

	"github.com/keonho-kim/meca/internal/agent/inference"
	"github.com/keonho-kim/meca/internal/agent/lat"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

var errReviewSuperseded = errors.New("independent review superseded by new user instructions")

func (s *execution) reviewInstructions() error {
	if !s.review.active || s.runner.PollSteering == nil {
		return nil
	}
	if messages := s.runner.PollSteering(); len(messages) > 0 {
		s.review.steering = append(s.review.steering, messages...)
		return errReviewSuperseded
	}
	return nil
}

func (s *execution) reviewWork(ctx context.Context, draft modelapi.Response) (bool, error) {
	if !s.runner.LAT || s.review.reviewed || strings.TrimSpace(draft.Text) == "" {
		return false, nil
	}
	s.review.reviewed = true // Feedback repairs do not start a recursive review cycle.
	// Settle the completed main reply before the side conversation can replace
	// its live stream. This same message is the canonical draft on every exit.
	if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: draft.Text, Reasoning: draft.Reasoning, ReasoningState: draft.ReasoningState}); err != nil {
		return false, err
	}
	s.review.draftSaved = true
	if err := s.emit(runrecord.Event{Kind: "progress", Text: draft.Text}); err != nil {
		return false, err
	}
	if err := s.emit(runrecord.Event{Kind: "lat_started"}); err != nil {
		return false, err
	}
	review, err := s.independentReview(ctx, draft.Text)
	s.checkpoint.ModelCalls = s.usage.snapshot().ModelCalls
	if ctx.Err() != nil {
		return false, errors.Join(ctx.Err(), err)
	}
	if errors.Is(err, errRuntimeRecord) {
		return false, err
	}
	if len(s.review.steering) > 0 {
		messages := s.review.steering
		s.review.steering = nil
		s.review.draftSaved = false
		if err := s.applyInstructions(ctx, messages); err != nil {
			return false, err
		}
		return true, s.reviewEvent(ctx, "lat_discarded", "")
	}
	// A review may take time. Admit new user instructions before its feedback
	// can cause another model decision, and never act on a stale request.
	if changed, steerErr := s.steer(ctx); changed || steerErr != nil {
		s.review.draftSaved = false
		return changed, errors.Join(steerErr, s.reviewEvent(ctx, "lat_discarded", ""))
	}
	if err != nil {
		return false, s.reviewEvent(ctx, "lat_unavailable", s.runner.Redact(err.Error()))
	}
	if review.Clean {
		return false, s.reviewEvent(ctx, "lat_clean", "")
	}
	feedback := s.runner.Redact(review.Feedback)
	if err := s.record(ctx, runrecord.Event{Kind: "lat_feedback", Text: feedback}); err != nil {
		return false, err
	}
	s.messages = append(s.messages, lat.Feedback(feedback))
	s.review.draftSaved = false
	return true, s.emit(runrecord.Event{Kind: "lat_feedback", RunID: s.checkpoint.ID, Text: feedback})
}

func (s *execution) independentReview(ctx context.Context, draft string) (lat.Result, error) {
	s.review.active = true
	defer func() { s.review.active = false }()
	text, err := s.runLATPhase(ctx, "review", s.latTemplates().LAT, inference.LAT, lat.Input{Plan: s.review.plan, Draft: s.runner.Redact(draft)})
	if err != nil {
		return lat.Result{}, err
	}
	return lat.Parse(text)
}

func (s *execution) reviewEvent(ctx context.Context, kind, text string) error {
	event := runrecord.Event{Kind: kind, RunID: s.checkpoint.ID, Text: text}
	if err := s.record(ctx, event); err != nil {
		return err
	}
	return s.emit(event)
}
