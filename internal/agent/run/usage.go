// Purpose: Record measured usage per attempt and apply retry budgets before restarting generation.
// Pattern: Model I/O boundary.
// Usage: Normal execution generation and compaction both call execution.generate.
// Related: internal/agent/run/generate.go, internal/agent/run/memory.go, internal/agent/run/statistics.go, internal/storage/sqlite/usage.go.
package run

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"time"
)

func (s *execution) generate(ctx context.Context, request modelapi.Request, account bool, consume func(modelapi.Event) error) error {
	if err := s.checkTimeBudget(ctx); err != nil {
		return err
	}
	request.IdleTimeout = time.Duration(s.limits.MaxModelSeconds) * time.Second
	var usage *modelapi.TokenUsage
	var rate *float64
	var measured time.Duration
	started := time.Now()
	role := inference.Agent
	if !account {
		role = inference.Compaction
	}
	generation := Generation{ID: "generation_" + rand.Text(), Role: role, Attempt: 1}
	if err := s.emit(Event{Kind: "generation_started"}); err != nil {
		return err
	}
	var outputBytes int64
	received := false
	flush := func() error {
		flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		generation.ElapsedMillis = time.Since(started).Milliseconds()
		s.statistics.add(usage, measured)
		err := s.recordUsage(flushCtx, usage, rate, generation)
		if account && (usage != nil || received) {
			err = errors.Join(err, s.countRequest(flushCtx, request, usage, outputBytes))
		}
		usage = nil
		rate = nil
		measured = 0
		outputBytes = 0
		received = false
		return err
	}
	err := s.runner.Generator.Generate(ctx, request, func(event modelapi.Event) error {
		if event.Retry != nil {
			if err := flush(); err != nil {
				return err
			}
			if !s.modelCall() {
				return fmt.Errorf("%w before provider retry: %s", errModelBudget, s.runner.Redact(event.Retry.Reason))
			}
			s.checkpoint.ModelCalls = s.usage.snapshot().ModelCalls
			message := fmt.Sprintf("Provider retry %d/%d: %s", event.Retry.Attempt, event.Retry.MaxAttempts, s.runner.Redact(event.Retry.Reason))
			if err := s.emit(Event{Kind: "retry", Text: message}); err != nil {
				return err
			}
			started = time.Now()
			generation.Attempt++
			if err := s.emit(Event{Kind: "generation_started"}); err != nil {
				return err
			}
			return consume(event)
		}
		if event.Delta != "" || event.Thinking != "" || len(event.Calls) > 0 || event.Done {
			received = true
		}
		outputBytes += int64(len(event.Delta) + len(event.Thinking))
		for _, call := range event.Calls {
			outputBytes += int64(len(call.ID) + len(call.Name) + len(call.Arguments))
		}
		if event.Usage != nil {
			if err := event.Usage.Check(); err != nil {
				return err
			}
			snapshot := *event.Usage
			usage = &snapshot
			measured = time.Since(started)
			rate = outputRate(snapshot.Output, measured)
			if err := s.emit(Event{Kind: "usage_progress", Usage: usage, TokensPerSecond: rate}); err != nil {
				return err
			}
		}
		return consume(event)
	})
	return errors.Join(err, flush())
}

func (s *execution) recordUsage(ctx context.Context, usage *modelapi.TokenUsage, rate *float64, generation Generation) error {
	// Preserve the last received count even if the call ends without another report.
	flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	final := Event{Kind: "generation_finished", Generation: &generation}
	if usage != nil {
		final.Kind = "usage"
		final.Usage = usage
	}
	if saveErr := s.runner.Journal.RecordEvent(flushCtx, s.runner.Session, final); saveErr != nil {
		return errors.Join(errRuntimeRecord, saveErr)
	}
	final.TokensPerSecond = rate
	return s.emit(final)
}
