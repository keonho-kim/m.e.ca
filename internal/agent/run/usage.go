// Purpose: Record measured usage per attempt and apply retry budgets before restarting generation.
// Pattern: Model I/O boundary.
// Usage: Agent and compaction share execution.generate accounting.
// Related: internal/agent/run/generate.go, internal/agent/run/memory.go, internal/agent/run/statistics.go, internal/storage/sqlite/usage.go.
package run

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/presentation"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *execution) generate(ctx context.Context, request modelapi.Request, role inference.Role, send modelapi.Stream, consume func(modelapi.Event) error) error {
	if err := s.checkTimeBudget(ctx); err != nil {
		return err
	}
	if s.order == nil {
		_, s.order = presentation.Scope(ctx)
	}
	if send == nil {
		request.IdleTimeout = s.modelIdleTimeout()
		send = func(ctx context.Context, emit func(modelapi.Event) error) error {
			return s.runner.Generator.Generate(ctx, request, emit)
		}
	}
	var usage *modelapi.TokenUsage
	var measured time.Duration
	started := time.Now()
	generation := runrecord.Generation{ID: "generation_" + rand.Text(), Role: role, Attempt: 1}
	// Open one presentation block per model decision. Usage still records
	// each attempt, including retries and compaction, under its own identity.
	if role == inference.Agent && s.presentation == "" {
		s.presentation = s.order.Next()
		copy := generation
		if err := s.emit(runrecord.Event{Kind: "generation_started", Generation: &copy}); err != nil {
			return err
		}
		if err := s.emit(runrecord.Event{Kind: "stream"}); err != nil {
			return err
		}
	}
	var outputBytes int64
	var inputBytes int64
	var budget modelapi.InputBudget
	var metrics attemptMetrics
	startAttempt := func() error {
		metrics = attemptMetrics{activity: runrecord.ModelActivity{Started: started, Role: role}}
		copy := metrics.activity
		return s.emit(runrecord.Event{Kind: "model_activity", Activity: &copy})
	}
	if err := startAttempt(); err != nil {
		return err
	}
	if role == inference.Agent {
		generation.BeforeCompaction = s.beforeCompaction
		s.beforeCompaction = nil
	}
	open := true
	received, completed := false, false
	flush := func() error {
		if !open {
			return nil
		}
		open = false
		flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		generation.ElapsedMillis = time.Since(started).Milliseconds()
		generation.MeasuredMillis = measured.Milliseconds()
		s.statistics.add(usage, measured)
		generation.Context = measuredContext(request, budget, inputBytes, usage)
		generation.FirstOutputMillis = nil
		if !metrics.activity.FirstOutput.IsZero() {
			generation.FirstOutputMillis = new(metrics.activity.FirstOutput.Sub(started).Milliseconds())
		}
		generation.Speed = nil
		if metrics.activity.Speed.Source != "" {
			generation.Speed = new(metrics.activity.Speed)
		}
		activity := metrics.activity
		activity.Finished = true
		err := s.recordUsage(flushCtx, usage, measured, generation, &activity)
		if role == inference.Agent && (usage != nil || received) {
			err = errors.Join(err, s.countRequest(flushCtx, request, usage, budget.InputTokens, inputBytes, outputBytes))
		}
		usage = nil
		measured = 0
		outputBytes = 0
		received = false
		return err
	}
	err := s.observeModelIO(ctx, send, func(event modelapi.Event) error {
		if event.InputBytes > 0 {
			inputBytes = event.InputBytes
		}
		if event.InputBudget != nil {
			budget = *event.InputBudget
		}
		if event.Retry != nil {
			reason := s.runner.Redact(event.Retry.Reason)
			if completed {
				return faults.Errorf(diagcatalog.RunProviderRetryAfterCompletion, reason)
			}
			if err := s.checkTimeBudget(ctx); err != nil {
				return err
			}
			if err := flush(); err != nil {
				return err
			}
			if !s.modelCall() {
				return faults.Errorf(diagcatalog.RunBeforeProviderRetry, errModelBudget, reason)
			}
			s.checkpoint.ModelCalls = s.usage.ModelCalls
			message := fmt.Sprintf("Provider retry %d/%d: %s", event.Retry.Attempt, event.Retry.MaxAttempts, reason)
			if err := s.emit(runrecord.Event{Kind: "retry", Text: message, ErrorCode: diagcatalog.Text(diagcatalog.RunProviderRetry)}); err != nil {
				return err
			}
			started = time.Now()
			generation.Attempt++
			open = true
			if err := startAttempt(); err != nil {
				return err
			}
			return consume(event)
		}
		if event.Usage != nil {
			if err := event.Usage.Check(); err != nil {
				return err
			}
		}
		var now time.Time
		if metrics.activity.FirstOutput.IsZero() || event.Usage != nil {
			now = time.Now()
			if metrics.observe(event, now) {
				copy := metrics.activity
				if err := s.emit(runrecord.Event{Kind: "model_activity", Activity: &copy}); err != nil {
					return err
				}
			}
		}
		if event.ReplaceOutput {
			outputBytes = 0
		}
		if event.HasOutput() {
			received = true
		}
		completed = completed || event.Done
		outputBytes += int64(len(event.Delta) + len(event.Thinking))
		for _, call := range event.Calls {
			outputBytes += int64(len(call.ID) + len(call.Name) + len(call.Arguments))
		}
		if event.Usage != nil {
			if usage != nil && usage.Equal(*event.Usage) {
				return consume(event)
			}
			snapshot := event.Usage.Clone()
			// Identical output counts, including completion or cost-only updates,
			// do not extend the interval in which those tokens were observed.
			if usage == nil || usage.Output != snapshot.Output {
				measured = now.Sub(started)
			}
			usage = &snapshot
			activity := metrics.activity
			if err := s.emit(runrecord.Event{Activity: &activity, Kind: "usage_progress", Usage: usage, MeasuredDuration: measured}); err != nil {
				return err
			}
		}
		return consume(event)
	})
	return errors.Join(err, flush())
}

func (s *execution) recordUsage(ctx context.Context, usage *modelapi.TokenUsage, measured time.Duration, generation runrecord.Generation, activity *runrecord.ModelActivity) error {
	// Preserve the last received count even if the call ends without another report.
	flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	final := runrecord.Event{Kind: "generation_finished", Generation: &generation}
	final.RunID, final.OccurredAt = s.checkpoint.ID, time.Now().UTC().Format(time.RFC3339Nano)
	if usage != nil {
		final.Kind = "usage"
		final.Usage = usage
	}
	if saveErr := s.runner.Journal.RecordEvent(flushCtx, s.runner.Session, final); saveErr != nil {
		return errors.Join(errRuntimeRecord, saveErr)
	}
	final.Activity = activity
	final.MeasuredDuration = measured
	return s.emit(final)
}
