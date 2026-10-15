// Purpose: Schedule validated native calls in bounded independent waves and preserve paired results.
// Pattern: Serial coordinator with bounded parallel execution.
// Usage: Run dispatches multi-call responses here; one owner mutates runtime and journal state.
// Related: internal/agent/run/batch_wave.go, internal/agent/run/call_validation.go.
package run

import (
	"context"
	"errors"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"time"
)

const maxParallelTools = 4

// An executor opts in only when it can resolve aliases and reason about actual accesses.
type parallelExecutor interface {
	CanParallel(context.Context, policy.Settings, toolapi.Call, toolapi.Call) (bool, error)
}

func (s *execution) batch(ctx context.Context, reply modelapi.Response, approve Approver) error {
	if s.usage.Iterations+len(reply.Calls)-1 > s.limits.MaxIterations {
		return s.returnToolFeedback(ctx, reply, "Batch exceeds the remaining action budget; submit fewer calls. Nothing executed.")
	}
	calls := make([]validatedCall, len(reply.Calls))
	for i, call := range reply.Calls {

		value, err := s.validateCall(call)
		if err != nil {
			return s.returnToolFeedback(ctx, reply, "Batch validation failed; nothing executed: "+err.Error())
		}
		calls[i] = value
	}
	s.usage.Iterations += len(calls) - 1
	s.checkpoint.Iteration = s.usage.Iterations
	if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: reply.Text, Reasoning: reply.Reasoning, Calls: reply.Calls}); err != nil {
		return err
	}
	if reply.Text != "" {
		if err := s.emit(Event{Kind: "progress", Text: reply.Text}); err != nil {
			return err
		}
	}
	for start := 0; start < len(calls); {
		end, err := s.waveEnd(ctx, calls, start)
		if err != nil {
			return errors.Join(s.skipCalls(ctx, calls[start:], err.Error()), fatalBatchError(ctx, err))
		}
		failed, err := s.batchWave(ctx, calls[start:end], approve)
		if failed || err != nil {
			skipErr := s.skipCalls(ctx, calls[end:], "Earlier batch operation failed or was cancelled; inspect recorded outcomes before retrying.")
			return errors.Join(err, skipErr)
		}
		start = end
	}
	return nil
}

func (s *execution) waveEnd(ctx context.Context, calls []validatedCall, start int) (int, error) {
	planner, ok := s.runner.Tools.(parallelExecutor)
	if !ok {
		return start + 1, nil
	}
	end := start + 1
	for end < len(calls) && end-start < maxParallelTools {
		for i := start; i < end; i++ {
			independent, err := planner.CanParallel(ctx, s.runner.Settings, calls[i].call, calls[end].call)
			if err != nil {
				return end, err
			}
			if !independent {
				return end, nil
			}
		}
		end++
	}
	return end, nil
}

func fatalBatchError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, errApprovalDenied) || errors.Is(err, errRuntimeRecord) {
		return err
	}
	return nil
}

func (s *execution) skipCalls(ctx context.Context, calls []validatedCall, detail string) error {
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	var result error
	for _, v := range calls {
		failure := &toolapi.Failure{Code: "batch_not_executed", Detail: detail, Next: "Inspect completed calls; submit only work that remains necessary."}
		outcome := toolapi.Result{Failed: true, Failure: failure, Content: []byte(`{"executed":false}`)}
		event := Event{Kind: "tool_failed", CallID: v.call.ID, Call: &v.call, Text: failure.Error(), Output: outcome.Content}
		if err := s.record(recordCtx, event); err != nil {
			return errors.Join(result, err)
		}
		if err := s.emit(event); err != nil {
			result = errors.Join(result, err)
		}
		if err := s.acceptResult(recordCtx, v, outcome, failure); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
}

func (s *execution) activateCall(ctx context.Context, v validatedCall) error {
	s.checkpoint.Tool, s.checkpoint.Arguments = v.call.Name, v.arguments
	if err := s.ground(ctx, v.call, v.reason); err != nil {
		return errors.Join(errRuntimeRecord, err)
	}
	if err := s.checkRepeated(v); err != nil {
		return err
	}
	if err := s.record(ctx, Event{Kind: "tool_proposed", CallID: v.call.ID, Call: &v.call, Text: "Grounded native candidate; not executed"}); err != nil {
		return errors.Join(errRuntimeRecord, err)
	}
	return nil
}
