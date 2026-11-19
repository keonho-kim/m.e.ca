// Purpose: Prepare the next agent request against the admitted conversation and provider limits.
// Pattern: Execution-owned preparation boundary.
// Usage: The decision loop discards stale or compacted preparations before starting generation.
// Related: internal/agent/run/loop.go, internal/agent/run/context.go, internal/provider/httpclient/run_generator.go.
package run

import (
	"context"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type requestPreparer interface {
	Prepare(context.Context, modelapi.Request) (modelapi.PreparedGeneration, error)
}

type decisionRequest struct {
	request                              modelapi.Request
	prepared                             modelapi.PreparedGeneration
	commandRevision, instructionRevision uint64
	supplied                             int
}

func (s *execution) prepareAgentRequest(ctx context.Context) (decisionRequest, error) {
	if err := s.restoreHistory(ctx); err != nil {
		return decisionRequest{}, err
	}
	if s.runner.BeforeDecision != nil {
		s.runner.BeforeDecision()
	}
	request, err := s.request(ctx)
	if err != nil {
		return decisionRequest{}, err
	}
	request.IdleTimeout = s.modelIdleTimeout()
	decision := decisionRequest{
		request: request, supplied: len(s.messages), commandRevision: s.commandRevision,
		instructionRevision: s.instructionRevision,
	}
	if provider, ok := s.runner.Generator.(requestPreparer); ok {
		prepareCtx, cancel := context.WithDeadline(ctx, s.usage.deadline)
		defer cancel()
		err = s.observeModelIO(prepareCtx, func(ctx context.Context, _ func(modelapi.Event) error) error {
			prepared, err := provider.Prepare(ctx, request)
			decision.prepared = prepared
			return err
		}, nil)
	}
	return decision, err
}

func (s *execution) modelIdleTimeout() time.Duration {
	return time.Duration(s.limits.MaxModelSeconds) * time.Second
}

func (s *execution) preparedCurrent(ctx context.Context, decision decisionRequest) (bool, error) {
	if decision.prepared.Send == nil {
		return true, nil
	}
	// Tokenization can outlast background commands and steering. A prepared
	// request describes exactly the conversation it was measured against.
	changed, err := s.admitInstructions(ctx)
	if err != nil || changed {
		return false, err
	}
	return decision.commandRevision == s.commandRevision &&
		decision.instructionRevision == s.instructionRevision &&
		decision.supplied == len(s.messages), nil
}
