// Purpose: Return rejected native candidates as paired tool results without performing side effects.
// Pattern: Native tool dialogue recovery.
// Usage: The decision loop commits rejected calls and a bounded correction before the next model request.
// Related: internal/agent/run/codemode.go, internal/agent/run/journal.go, internal/agent/context/conversation.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const maxArgumentCorrections = 2

func (s *execution) rejectInvalidCalls(ctx context.Context, reply modelapi.Response, problems []error, corrections int) error {
	var first string
	for i, problem := range problems {
		if problem != nil {
			first = fmt.Sprintf("%s (%s): %v", reply.Calls[i].Name, reply.Calls[i].ID, problem)
			break
		}
	}
	if err := s.returnToolFeedback(ctx, reply, first, problems); err != nil {
		return err
	}
	if corrections >= maxArgumentCorrections {
		return faults.New(diagcatalog.FailedPrecondition, diagcatalog.RunToolArgumentCorrectionExhausted, maxArgumentCorrections, s.runner.Redact(first))
	}
	if s.usage.ModelCalls >= s.limits.MaxModelCalls {
		// The decision loop owns budget termination. There is no retry to announce.
		return nil
	}
	return s.emit(runrecord.Event{Kind: "retry", Text: diagcatalog.Text(diagcatalog.RunRetryInvalidToolCall)})
}

func (s *execution) returnToolFeedback(ctx context.Context, reply modelapi.Response, detail string, problems []error) error {
	calls := append([]toolapi.Call(nil), reply.Calls...)
	if len(calls) == 0 || len(calls) > toolapi.MaxCalls {
		return faults.Errorf(diagcatalog.RunInvalidNativeCallCount)
	}
	for i, call := range calls {
		calls[i] = s.identifyCall(call)
		if err := call.Check(); err != nil {
			return err
		}
	}
	assistant := modelapi.Message{Role: "assistant", Content: reply.Text, Reasoning: reply.Reasoning, ReasoningState: reply.ReasoningState, Calls: calls, PreserveObservations: true}
	records := make([]runrecord.Entry, 0, 2*len(calls)+2)
	records = append(records, runrecord.Entry{Message: &assistant})
	events := make([]runrecord.Event, len(calls))
	receipts := make([]runrecord.EventReference, len(calls))
	for i, call := range calls {
		feedback := detail
		var field string
		if problems != nil {
			if problem := problems[i]; problem != nil {
				feedback = problem.Error()
				var invalid *toolapi.ArgumentError
				if errors.As(problem, &invalid) {
					field = invalid.Name
				}
			} else {
				feedback = "Skipped because another call in this batch had invalid arguments. Resubmit this call if it is still needed."
			}
		}
		feedback = s.runner.Redact(feedback)
		body, err := json.Marshal(struct {
			CallID   string `json:"call_id"`
			Tool     string `json:"tool"`
			Field    string `json:"field,omitempty"`
			Feedback string `json:"feedback"`
			Executed bool   `json:"executed"`
		}{CallID: call.ID, Tool: call.Name, Field: field, Feedback: feedback})
		if err != nil {
			return err
		}
		message := modelapi.Message{Role: "tool", Content: string(body), ToolCallID: call.ID, ToolName: call.Name}
		events[i] = runrecord.Event{Kind: "tool_failed", Text: "Not executed: " + feedback, CallID: call.ID, Call: &calls[i], Output: body}
		records = append(records, runrecord.Entry{Message: &message}, runrecord.Entry{Event: &events[i], Receipt: &receipts[i]})
	}
	s.result.HadToolFailure = true
	s.checkpoint.Observation = s.runner.Redact(detail)
	checkpoint, err := s.checkpointEvent()
	if err != nil {
		return err
	}
	records = append(records, runrecord.Entry{Event: &checkpoint})
	if err := s.saveEntries(ctx, records); err != nil {
		return err
	}
	for i, event := range events {
		if receipts[i].Sequence > 0 {
			event.Reference = &receipts[i]
		}
		if err := s.emit(event); err != nil {
			return err
		}
	}
	return nil
}
