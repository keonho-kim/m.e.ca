// Purpose: Gate a frozen file candidate on a single semantic review before user approval.
// Pattern: Review orchestration with run-local rejection reuse.
// Usage: execute calls reviewCandidate after scope and snapshot preparation.
// Related: internal/agent/conclave/review.go, internal/agent/run/review_context.go.
package run

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/agent/verification"
)

const reviewRejected = "conclave_rejected"
const reviewFailed = "conclave_error"

func (s *execution) reviewCandidate(ctx context.Context, call toolapi.Call, proposal toolapi.Approval) (*conclave.Result, error) {
	if proposal.Candidate == nil {
		if call.Name == "write_file" || call.Name == "edit_file" {
			return nil, fmt.Errorf("content change has no frozen candidate")
		}
		return nil, nil
	}
	candidate := *proposal.Candidate
	if err := candidate.Check(call); err != nil {
		return nil, err
	}
	if candidate.Lines() == 0 {
		return nil, nil
	}
	s.checkpoint.Phase = "conclave"
	if err := s.persist(ctx); err != nil {
		return nil, err
	}
	parameters, err := inference.Resolve(s.runner.Inference, inference.Conclave)
	if err != nil {
		return nil, err
	}
	instructions, err := prompt.Internal(s.runner.templates().Conclave, s.runner.ProjectInstructions)
	if err != nil {
		return nil, err
	}
	keyBody, err := json.Marshal(struct {
		Files                        []toolapi.FileDelta
		Request, Instructions, Model string
		Evidence                     int
		Checks                       []verification.Record
	}{candidate.Files, s.checkpoint.Objective, instructions, s.runner.Model, s.evidenceRevision, s.checkpoint.Checks})
	if err != nil {
		return nil, err
	}
	key := sha256.Sum256(keyBody)
	result, cached := s.reviews[key]
	if cached {
		result.CallID = call.ID
		result.CandidateDigest, err = candidate.Digest()
		if err != nil {
			return nil, err
		}
		result.Cached = true
	} else {
		request := parameters.Apply(modelapi.Request{ModelID: s.runner.Model, Thinking: s.runner.Thinking, ReasoningEffort: s.runner.ReasoningEffort, Messages: []modelapi.Message{{Role: "system", Content: instructions}}})
		request = parameters.ApplyThinking(request, s.runner.ThinkingSupported)
		result, err = conclave.Review(ctx, conclave.Input{Checks: s.checkpoint.Checks, Request: s.checkpoint.Objective, Grounding: *s.checkpoint.Grounding, Call: call, Candidate: candidate}, request, s.limits.MaxContextBytes, s.reviewGenerate, s.reviewContext, s.runner.Redact)
		if err != nil {
			return nil, err
		}
		if result.Complete && result.Verdict.Decision == "reject" {
			if s.reviews == nil || len(s.reviews) >= 32 {
				s.reviews = map[[32]byte]conclave.Result{}
			}
			s.reviews[key] = result
		}
	}
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
		return &result, &toolapi.Failure{Code: code, Detail: result.Feedback(), Next: "Correct the specific defect or provide new evidence; the unchanged candidate will not be reviewed again."}
	}
	return &result, nil
}
