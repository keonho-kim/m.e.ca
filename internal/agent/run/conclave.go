// Purpose: Gate prepared file changes on a durable Conclave decision before user approval.
// Pattern: Review use case.
// Usage: execute invokes reviewCandidate against the exact prepared candidate and current Monitor snapshot.
// Related: internal/agent/conclave/review.go, internal/agent/run/execute.go, internal/agent/run/conclave_usage.go.
package run

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

const reviewRejected = "conclave_rejected"
const reviewFailed = "conclave_error"

func (s *task) reviewCandidate(ctx context.Context, call toolapi.Call, proposal toolapi.Approval) (*conclave.Result, error) {
	if proposal.Candidate == nil {
		if call.Name == "write_file" || call.Name == "edit_file" {
			return nil, fmt.Errorf("content-changing tool has no frozen review candidate")
		}
		return nil, nil
	}
	candidate := *proposal.Candidate
	if err := candidate.Check(call); err != nil {
		return nil, err
	}
	n, required := conclave.Allocation(candidate.Lines())
	if n == 0 {
		return nil, nil
	}
	s.checkpoint.Phase = "conclave"
	if err := s.persist(ctx); err != nil {
		return nil, err
	}
	history, err := s.runner.observer.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	digest, err := candidate.Digest()
	if err != nil {
		return nil, err
	}
	unavailable := func(cause error) conclave.Result {
		result := conclave.Result{CallID: call.ID, CandidateDigest: digest, Lines: candidate.Lines(), Reviews: n, Required: required, Votes: make([]conclave.Vote, n)}
		for i := range result.Votes {
			result.Votes[i] = conclave.Vote{Index: i + 1, Error: "Review could not complete: " + excerpt(s.runner.Redact(cause.Error()), 1000)}
		}
		return result
	}
	var result conclave.Result
	if s.usage.ModelCalls+n > s.limits.MaxModelCalls {
		result = unavailable(fmt.Errorf("insufficient remaining model calls for %d reviewers", n))
	} else {
		s.usage.ModelCalls += n
		account := &reviewerUsage{task: s, remaining: int64(s.limits.MaxModelCalls - s.usage.ModelCalls), active: make([]modelapi.TokenUsage, n)}
		input := conclave.Input{TaskObjective: s.checkpoint.Objective, Monitor: history, Grounding: *s.checkpoint.Grounding, Call: call, Candidate: candidate}
		if s.runner.assignment != nil {
			input.GoalObjective = s.runner.assignment.GoalObjective
		}
		parameters, err := inference.Resolve(s.runner.Inference, inference.Conclave)
		if err != nil {
			return nil, err
		}
		reviewSystem, err := prompt.Internal(s.runner.templates().Conclave, s.runner.ProjectInstructions)
		if err != nil {
			return nil, err
		}
		request := modelapi.Request{ModelID: s.runner.Model, Thinking: s.runner.Thinking, ReasoningEffort: s.runner.ReasoningEffort, Messages: []modelapi.Message{{Role: "system", Content: reviewSystem}}}
		result, err = conclave.Review(ctx, input, parameters.Apply(request), s.limits.MaxContextBytes, account.generate, s.runner.Redact)
		s.usage.ModelCalls += int(account.extra.Load()) - (n - int(account.started.Load()))
		if err != nil && ctx.Err() == nil {
			result = unavailable(err)
		}
	}
	s.checkpoint.ModelCalls = s.usage.ModelCalls
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err := result.Check(); err != nil {
		return nil, err
	}
	s.checkpoint.Conclave = &result
	s.checkpoint.PendingReview = !result.Passed
	event := Event{Kind: "conclave", CallID: call.ID, Call: &call, Conclave: &result}
	if err := s.record(ctx, event); err != nil {
		return nil, err
	}
	if err := s.emit(event); err != nil {
		return nil, err
	}
	if err := s.persist(ctx); err != nil {
		return nil, err
	}
	if !result.Passed {
		code := reviewRejected
		if !result.Complete {
			code = reviewFailed
		}
		return &result, &toolapi.Failure{Code: code, Detail: result.Feedback(), Next: "Generate a new grounded PFC candidate within the existing Goal budget. No separate rejection-count limit applies."}
	}
	return &result, nil
}
