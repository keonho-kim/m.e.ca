// Purpose: Run one optional independent review before finishing observed tool work.
// Pattern: Request-owned optional phase with advisory feedback.
// Usage: The normal loop calls reviewWork after steering and active commands settle.
// Related: internal/agent/lat/packet.go, internal/agent/run/usage.go, internal/agent/run/finish.go.
package run

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/keonho-kim/meca/internal/agent/inference"
	"github.com/keonho-kim/meca/internal/agent/lat"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

var errReviewSuperseded = errors.New("independent review superseded by new user instructions")

func (s *execution) reviewInstructions() error {
	if !s.reviewing || s.runner.PollSteering == nil {
		return nil
	}
	if messages := s.runner.PollSteering(); len(messages) > 0 {
		s.latSteering = append(s.latSteering, messages...)
		return errReviewSuperseded
	}
	return nil
}

func (s *execution) reviewWork(ctx context.Context, draft modelapi.Response) (bool, error) {
	if !s.runner.LAT || s.latReviewed || !s.latObserved || strings.TrimSpace(draft.Text) == "" {
		return false, nil
	}
	s.latReviewed = true // Feedback repairs do not start a recursive review cycle.
	if err := s.emit(runrecord.Event{Kind: "lat_started"}); err != nil {
		return false, err
	}
	review, err := s.independentReview(ctx, draft.Text)
	s.checkpoint.ModelCalls = s.usage.snapshot().ModelCalls
	if ctx.Err() != nil {
		flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		// The main reply completed before review began, but was not committed
		// as a final answer. Preserve that visible draft for interruption/resume.
		saveErr := s.save(flush, modelapi.Message{Role: "assistant", Content: draft.Text, Reasoning: draft.Reasoning, Incomplete: true})
		return false, errors.Join(ctx.Err(), err, saveErr)
	}
	if errors.Is(err, errRuntimeRecord) {
		return false, err
	}
	if len(s.latSteering) > 0 {
		messages := s.latSteering
		s.latSteering = nil
		if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: draft.Text, Reasoning: draft.Reasoning, ReasoningState: draft.ReasoningState}); err != nil {
			return false, err
		}
		if err := s.emit(runrecord.Event{Kind: "progress", Text: draft.Text}); err != nil {
			return false, err
		}
		if err := s.applyInstructions(ctx, messages); err != nil {
			return false, err
		}
		return true, s.reviewEvent(ctx, "lat_discarded", "")
	}
	// A review may take time. Admit new user instructions before its feedback
	// can cause another model decision, and never act on a stale request.
	if changed, steerErr := s.continueSteered(ctx, draft); changed || steerErr != nil {
		return changed, errors.Join(steerErr, s.reviewEvent(ctx, "lat_discarded", ""))
	}
	if err != nil {
		return false, s.reviewEvent(ctx, "lat_unavailable", s.runner.Redact(err.Error()))
	}
	if review.Clean {
		return false, s.reviewEvent(ctx, "lat_clean", "")
	}
	feedback := s.runner.Redact(review.Feedback)
	if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: draft.Text, Reasoning: draft.Reasoning, ReasoningState: draft.ReasoningState}); err != nil {
		return false, err
	}
	if err := s.emit(runrecord.Event{Kind: "progress", Text: draft.Text}); err != nil {
		return false, err
	}
	if err := s.record(ctx, runrecord.Event{Kind: "lat_feedback", Text: feedback}); err != nil {
		return false, err
	}
	s.messages = append(s.messages, lat.Feedback(feedback))
	return true, s.emit(runrecord.Event{Kind: "lat_feedback", Text: feedback})
}

func (s *execution) independentReview(ctx context.Context, draft string) (lat.Result, error) {
	if s.usage.snapshot().ModelCalls >= s.limits.MaxModelCalls {
		return lat.Result{}, errModelBudget
	}
	parameters, err := inference.Resolve(s.runner.Inference, inference.LAT)
	if err != nil {
		return lat.Result{}, err
	}
	text := prompt.Defaults().LAT
	if s.runner.Prompts != nil && s.runner.Prompts.LAT != "" {
		text = s.runner.Prompts.LAT
	}
	input := lat.Input{
		Prompt: s.runner.Redact(text), Instructions: s.runner.Redact(s.runner.ProjectInstructions), Root: s.runner.ProjectRoot,
		Objective: s.checkpoint.Objective, Draft: s.runner.Redact(draft), ContextWindow: s.latWindow(), MaxOutput: parameters.MaxOutputTokens,
	}
	s.reviewing = true
	defer func() { s.reviewing = false }()
	input.Context, err = s.prepareLATContext(ctx, input)
	if err != nil {
		return lat.Result{}, err
	}
	request, err := lat.Build(input)
	if err != nil {
		return lat.Result{}, err
	}
	request.ModelID = s.runner.Model
	request.Temperature = &parameters.Temperature
	request.Thinking, request.ReasoningEffort = s.runner.Thinking, s.runner.ReasoningEffort
	request = parameters.ApplyThinking(request, s.runner.ThinkingSupported)
	if !s.modelCall() {
		return lat.Result{}, errModelBudget
	}
	ctx, cancel := context.WithDeadline(ctx, s.usage.snapshot().deadline)
	defer cancel()
	return lat.Review(ctx, func(ctx context.Context, request modelapi.Request, consume func(modelapi.Event) error) error {
		return s.generate(ctx, request, inference.LAT, consume)
	}, request)
}

func (s *execution) reviewEvent(ctx context.Context, kind, text string) error {
	event := runrecord.Event{Kind: kind, Text: text}
	if err := s.record(ctx, event); err != nil {
		return err
	}
	return s.emit(event)
}
