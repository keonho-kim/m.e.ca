// Purpose: Project runtime events into background-owned Monitor observations.
// Pattern: Observer adapter at the journal boundary.
// Usage: Runtime writes facts through record; Goal owners create and join their Monitor.
// Related: internal/agent/monitor/observer.go, internal/agent/run/events.go, internal/storage/sqlite/monitor.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/monitor"
)

func (r Runner) startMonitor(ctx context.Context) (*monitor.Observer, error) {
	for _, record := range r.MonitorSeed {
		if err := record.Check(); err != nil {
			return nil, err
		}
	}
	// This owner closes the worker after cancellation-safe terminal records are flushed.
	return monitor.New(context.WithoutCancel(ctx), r.MonitorSeed, func(ctx context.Context, record monitor.Record) error {
		return r.Journal.RecordEvent(ctx, r.Session, Event{Kind: "monitor", Monitor: &record})
	}), nil
}

func (s *task) record(ctx context.Context, event Event) error {
	if err := s.runner.Journal.RecordEvent(ctx, s.runner.Session, event); err != nil {
		return err
	}
	if s.runner.observer == nil {
		return fmt.Errorf("task Monitor is unavailable")
	}
	record := monitor.Record{TaskID: s.checkpoint.ID, GoalID: s.checkpoint.GoalID, CallID: event.CallID, Kind: event.Kind, Detail: event.Text}
	if event.Call != nil {
		record.Tool = event.Call.Name
		var target struct{ Path, Destination, Command string }
		if err := json.Unmarshal(event.Call.Arguments, &target); err != nil {
			return err
		}
		record.Detail = fmt.Sprintf("path=%s; destination=%s; command=%s; %s", target.Path, target.Destination, target.Command, record.Detail)
	}
	switch event.Kind {
	case "tool_proposed", "approved", "denied", "tool_started", "tool_succeeded", "tool_failed", "tool_interrupted":
	case "conclave":
		if event.Conclave == nil {
			return fmt.Errorf("missing Conclave observation")
		}
		record.Kind = "review_rejected"
		record.Detail = event.Conclave.Feedback()
		if event.Conclave.Passed {
			record.Kind = "review_passed"
			record.Detail = fmt.Sprintf("Approved %d/%d; required %d; candidate %s", event.Conclave.Approvals, event.Conclave.Reviews, event.Conclave.Required, event.Conclave.CandidateDigest)
		}
	case "task_checkpoint":
		if event.Checkpoint == nil {
			return nil
		}
		cp := event.Checkpoint
		if cp.Phase != "observation" && cp.Phase != "finished" && cp.Status == Running {
			return nil
		}
		record.Kind = "task_state"
		body, err := json.Marshal(struct {
			Status             Status
			Phase, Observation string
			Revision           int
		}{cp.Status, cp.Phase, cp.Observation, cp.ChangeRevision})
		if err != nil {
			return err
		}
		record.Detail = string(body)
	default:
		return nil
	}
	record.Detail = s.runner.Redact(record.Detail)
	return s.runner.observer.Observe(ctx, record)
}
