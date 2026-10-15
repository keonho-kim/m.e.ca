// Purpose: Record measured usage per attempt and apply retry budgets before restarting generation.
// Pattern: Model I/O boundary.
// Usage: Normal execution generation and compaction both call execution.generate.
// Related: internal/agent/run/generate.go, internal/agent/run/memory.go, internal/storage/sqlite/usage.go.
package run

import (
	"context"
	"errors"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"time"
)

func (s *execution) generate(ctx context.Context, request modelapi.Request, consume func(modelapi.Event) error) error {
	if err := s.checkTimeBudget(ctx); err != nil {
		return err
	}
	request.IdleTimeout = time.Duration(s.limits.MaxModelSeconds) * time.Second
	var usage *modelapi.TokenUsage
	err := s.runner.Generator.Generate(ctx, request, func(event modelapi.Event) error {
		if event.Retry != nil {
			if err := s.recordUsage(ctx, usage); err != nil {
				return err
			}
			usage = nil
			if s.usage.ModelCalls >= s.limits.MaxModelCalls {
				return fmt.Errorf("model call budget exhausted before provider retry: %s", s.runner.Redact(event.Retry.Reason))
			}
			s.usage.ModelCalls++
			s.checkpoint.ModelCalls = s.usage.ModelCalls
			message := fmt.Sprintf("Provider retry %d/%d: %s", event.Retry.Attempt, event.Retry.MaxAttempts, s.runner.Redact(event.Retry.Reason))
			if err := s.emit(Event{Kind: "retry", Text: message}); err != nil {
				return err
			}
			return consume(event)
		}
		if event.Usage != nil {
			if err := event.Usage.Check(); err != nil {
				return err
			}
			snapshot := *event.Usage
			usage = &snapshot
			if err := s.emit(Event{Kind: "usage_progress", Usage: usage}); err != nil {
				return err
			}
		}
		return consume(event)
	})
	return errors.Join(err, s.recordUsage(ctx, usage))
}

func (s *execution) recordUsage(ctx context.Context, usage *modelapi.TokenUsage) error {
	if usage == nil {
		return nil
	}
	// Preserve the last received count even if the call ends without another report.
	flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	final := Event{Kind: "usage", Usage: usage}
	if saveErr := s.runner.Journal.RecordEvent(flushCtx, s.runner.Session, final); saveErr != nil {
		return saveErr
	}
	return s.emit(final)
}
