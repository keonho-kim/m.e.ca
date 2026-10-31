// Purpose: Own one cancellable summary call and bounded measured-usage handoff.
// Pattern: Joined background worker with a coalesced progress snapshot.
// Usage: Controller starts one job; the runtime drains measurements and joins it.
// Related: internal/agent/lat/memory/controller.go, internal/agent/run/lat_memory_usage.go.
package memory

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type Receipt struct {
	ID                string
	Attempt           int
	Elapsed, Measured time.Duration
	Usage             *modelapi.TokenUsage
}

type Update struct {
	Receipts []Receipt
	Progress *Receipt
	Done     bool
}

type job struct {
	mu     sync.Mutex
	stop   context.CancelFunc
	done   chan struct{}
	update Update
	text   string
	err    error
}

func start(ctx context.Context, request modelapi.Request, generator modelapi.Generator, reserve func() bool) *job {
	child, stop := context.WithCancel(ctx)
	j := &job{stop: stop, done: make(chan struct{})}
	go j.run(child, request, generator, reserve)
	return j
}

func (j *job) run(ctx context.Context, request modelapi.Request, generator modelapi.Generator, reserve func() bool) {
	defer close(j.done)
	defer j.stop()
	var text strings.Builder
	completed := false
	started := time.Now()
	receipt := Receipt{ID: "generation_" + rand.Text(), Attempt: 1}
	active := true
	flush := func() {
		if !active {
			return
		}
		active = false
		receipt.Elapsed = time.Since(started)
		j.mu.Lock()
		j.update.Receipts = append(j.update.Receipts, receipt)
		j.update.Progress = nil
		j.mu.Unlock()
		receipt.Usage, receipt.Measured = nil, 0
	}
	err := generator.Generate(ctx, request, func(event modelapi.Event) error {
		if event.Retry != nil {
			flush()
			if !reserve() {
				return fmt.Errorf("LAT memory model-call budget exhausted before retry")
			}
			receipt.Attempt++
			active = true
			started = time.Now()
			text.Reset()
			completed = false
		}
		if len(event.Calls) > 0 {
			return fmt.Errorf("LAT memory summarization cannot call tools")
		}
		if event.Usage != nil {
			if err := event.Usage.Check(); err != nil {
				return err
			}
			usage := event.Usage.Clone()
			receipt.Usage, receipt.Measured = &usage, time.Since(started)
			progress := receipt
			j.mu.Lock()
			j.update.Progress = &progress
			j.mu.Unlock()
		}
		if text.Len()+len(event.Delta) > MaxBytes {
			return fmt.Errorf("LAT memory response exceeds 16 MiB")
		}
		text.WriteString(event.Delta)
		completed = completed || event.Done
		return ctx.Err()
	})
	flush()
	value := strings.TrimSpace(text.String())
	if err == nil && (!completed || value == "" || !utf8.ValidString(value)) {
		err = fmt.Errorf("LAT memory summarization did not produce a complete nonempty response")
	}
	j.mu.Lock()
	j.text, j.err, j.update.Done = value, errors.Join(err, ctx.Err()), true
	j.mu.Unlock()
}

func (j *job) drain() Update {
	j.mu.Lock()
	defer j.mu.Unlock()
	update := j.update
	j.update.Receipts, j.update.Progress = nil, nil
	return update
}
