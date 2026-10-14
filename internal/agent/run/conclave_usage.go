// Purpose: Account parallel reviewer attempts without racing the serial task budget or UI totals.
// Pattern: Parallel I/O accounting adapter.
// Usage: reviewCandidate reserves initial calls and invokes this adapter for each reviewer.
// Related: internal/agent/run/conclave.go, internal/agent/run/usage.go, internal/agent/conclave/review.go.
package run

import (
	"context"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"sync"
	"sync/atomic"
	"time"
)

type reviewerUsage struct {
	task           *task
	remaining      int64
	started, extra atomic.Int64
	mu             sync.Mutex
	active         []modelapi.TokenUsage
}

func (a *reviewerUsage) progress() error {
	total := modelapi.TokenUsage{}
	for _, usage := range a.active {
		total.Input += usage.Input
		total.Output += usage.Output
	}
	return a.task.emit(Event{Kind: "usage_progress", Usage: &total})
}
func (a *reviewerUsage) generate(ctx context.Context, request modelapi.Request, consume func(modelapi.Event) error) error {
	slot := int(a.started.Add(1) - 1)
	request.IdleTimeout = time.Duration(a.task.limits.MaxModelSeconds) * time.Second
	var usage *modelapi.TokenUsage
	bytes := 0
	flush := func() error {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.active[slot] = modelapi.TokenUsage{}
		err := a.task.recordUsage(ctx, usage)
		usage = nil
		return errors.Join(err, a.progress())
	}
	err := a.task.runner.Generator.Generate(ctx, request, func(event modelapi.Event) error {
		if event.Retry != nil {
			if err := flush(); err != nil {
				return err
			}
			for {
				used := a.extra.Load()
				if used >= a.remaining {
					return fmt.Errorf("model call budget exhausted before Conclave retry")
				}
				if a.extra.CompareAndSwap(used, used+1) {
					break
				}
			}
			bytes = 0
		}
		bytes += len(event.Delta) + len(event.Thinking)
		if event.Usage != nil {
			if err := event.Usage.Check(); err != nil {
				return err
			}
			copy := *event.Usage
			usage = &copy
			a.mu.Lock()
			a.active[slot] = copy
			err := a.progress()
			a.mu.Unlock()
			if err != nil {
				return err
			}
		}
		return consume(event)
	})
	count := compact.TokenCount{Amount: int64(bytes), Estimated: true, Kind: "model"}
	if usage != nil {
		count.Amount = usage.Output
		count.Estimated = false
	}
	a.mu.Lock()
	flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	observeErr := a.task.observe(flushCtx, count)
	cancel()
	a.mu.Unlock()
	return errors.Join(err, observeErr, flush())
}
