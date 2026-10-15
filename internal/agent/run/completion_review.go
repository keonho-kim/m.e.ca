// Purpose: Route final proposals and stalled work through a separate completion Conclave.
// Pattern: Run termination coordinator.
// Usage: The main loop calls finishOrReview; repetition detection calls judgeCompletion.
// Related: internal/agent/completion/review.go, internal/agent/run/report.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/completion"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"github.com/keonho-kim/v-code/internal/agent/response"
	"github.com/keonho-kim/v-code/internal/agent/reviewjson"
	"strings"
)

func (s *execution) finishOrReview(ctx context.Context, text string) (bool, error) {
	if (len(s.checkpoint.Work.Receipts) > 0 || s.result.HadToolFailure) && response.ToolMarkup(text) {
		if err := s.recordReviewAttempt(ctx, reviewjson.NewAttempt("main_protocol", 1, text, fmt.Errorf("non-native tool syntax"))); err != nil {
			return false, err
		}
		if s.textCorrection {
			_, err := s.report(ctx, "The model repeated textual tool syntax; no such tool was executed.", true)
			return true, err
		}
		s.textCorrection = true
		s.checkpoint.Observation = "Your text contained a tool invocation that was NOT executed. Use the supplied native function schemas for remaining work, or return a plain user-facing final answer. Do not emit XML or JSON tool invocations in assistant text."
		s.checkpoint.NoProgress++
		return false, s.persist(ctx)
	}
	if len(s.checkpoint.Work.Receipts) == 0 || strings.TrimSpace(text) == "" {
		return s.finish(ctx, text)
	}
	return s.judgeCompletion(ctx, text, "final answer proposal")
}
func (s *execution) judgeCompletion(ctx context.Context, draft, trigger string) (bool, error) {
	fingerprint := s.checkpoint.Work.Fingerprint(s.checkpoint.Objective)
	if fingerprint == s.checkpoint.Work.Reviewed {
		_, err := s.report(ctx, "No new evidence since the last completion review; stopping repeated work.", true)
		return true, err
	}
	if err := s.checkTimeBudget(ctx); err != nil {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		_, err = s.report(ctx, err.Error(), true)
		return true, err
	}
	if s.usage.ModelCalls >= s.limits.MaxModelCalls {
		_, err := s.report(ctx, "Completion review budget exhausted.", true)
		return true, err
	}
	instructions, err := prompt.Internal(s.runner.templates().Completion, s.runner.ProjectInstructions)
	if err != nil {
		return false, err
	}
	parameters, err := inference.Resolve(s.runner.Inference, inference.Completion)
	if err != nil {
		return false, err
	}
	request := parameters.Apply(modelapi.Request{ModelID: s.runner.Model, Thinking: s.runner.Thinking, ReasoningEffort: s.runner.ReasoningEffort, Messages: []modelapi.Message{{Role: "system", Content: instructions}}})
	request = parameters.ApplyThinking(request, s.runner.ThinkingSupported)
	s.checkpoint.Phase = "completion_review"
	if err := s.persist(ctx); err != nil {
		return false, err
	}
	input := completion.Input{PendingReview: s.checkpoint.PendingReview, Request: s.checkpoint.Objective, State: s.checkpoint.Work, Draft: excerpt(draft, 8192), Trigger: trigger}
	// Prior judgement is historical, not an instruction to the independent reviewer.
	input.State.Verdict = nil
	input.State.Reviewed = ""
	verdict, err := completion.Review(ctx, input, request, func(ctx context.Context, r modelapi.Request, consume func(modelapi.Event) error) error {
		return s.reviewGenerate(ctx, r, consume)
	}, s.runner.Redact, func(a reviewjson.Attempt) error { return s.recordReviewAttempt(ctx, a) })
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if errors.Is(err, errRuntimeRecord) {
		return false, err
	}
	if err != nil {
		_, reportErr := s.report(ctx, "Completion review failed: "+s.runner.Redact(err.Error()), true)
		return true, reportErr
	}
	body, _ := json.Marshal(verdict)
	if s.runner.Redact(string(body)) != string(body) {
		_, err := s.report(ctx, "Completion review contained protected content.", true)
		return true, err
	}
	s.checkpoint.Work.Reviewed = fingerprint
	s.checkpoint.Work.Verdict = &verdict
	if err := s.persist(ctx); err != nil {
		return false, err
	}
	event := Event{Kind: "completion_review", Internal: true, Text: string(body)}
	if err := s.record(ctx, event); err != nil {
		return false, err
	}
	switch verdict.Decision {
	case "complete":
		if draft != "" {
			return s.finish(ctx, draft)
		}
		_, err := s.report(ctx, verdict.Reason, false)
		return true, err
	case "blocked":
		_, err := s.report(ctx, verdict.Reason, true)
		return true, err
	default:
		s.checkpoint.NoProgress = 0
		s.checkpoint.Observation = fmt.Sprintf("Completion review: only these original requested outcomes remain unmet: %v. %s. Reuse existing results; do not restart or expand the request.", verdict.Unmet, verdict.Reason)
		// Keep rejected drafts out of assistant history; retain the review feedback.
		return false, s.persist(ctx)
	}
}
