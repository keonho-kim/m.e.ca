// Purpose: Collect completed commands on the runtime owner during provider preparation and generation.
// Pattern: Joined I/O worker with synchronous event handoff.
// Usage: Preparation and generation use this boundary while commands can finish independently.
// Related: internal/agent/run/commands.go, internal/agent/run/decision_request.go, internal/agent/run/usage.go.
package run

import (
	"context"
	"errors"
	"time"

	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (s *execution) observeModelIO(ctx context.Context, send modelapi.Stream, consume func(modelapi.Event) error) error {
	commands, ok := s.runner.Tools.(toolapi.CommandExecutor)
	if !ok || !commands.HasCommands() {
		return send(ctx, consume)
	}
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	var events chan modelapi.Event
	var reply chan error
	var emit func(modelapi.Event) error
	if consume != nil {
		// A single acknowledgement preserves serial backpressure and error ordering.
		events, reply = make(chan modelapi.Event), make(chan error, 1)
		emit = func(event modelapi.Event) error {
			select {
			case events <- event:
			case <-child.Done():
				return child.Err()
			}
			select {
			case err := <-reply:
				return err
			case <-child.Done():
				return child.Err()
			}
		}
	}
	done := make(chan error, 1)
	go func() { done <- faults.Guard(func() error { return send(child, emit) }) }()
	var updates <-chan struct{}
	if source, ok := s.runner.Tools.(toolapi.CommandNotifier); ok {
		updates = source.CommandUpdates()
	}
	var ticks <-chan time.Time
	if updates == nil {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		ticks = ticker.C
	}
	observe := func() error {
		_, err := s.collectCommands(ctx)
		return err
	}
	var failure error
	for {
		select {
		case message := <-events:
			if failure == nil {
				failure = consume(message)
			}
			reply <- failure
			if failure != nil {
				cancel()
			}
		case <-ticks:
			if failure == nil {
				failure = observe()
				if failure != nil {
					cancel()
				}
			}
		case <-updates:
			if failure == nil {
				failure = observe()
			}
			if failure != nil {
				cancel()
			}
		case err := <-done:
			if failure == nil && ctx.Err() == nil {
				failure = observe()
			}
			return errors.Join(err, failure)
		case <-ctx.Done():
			cancel()
			return errors.Join(ctx.Err(), <-done, failure)
		}
	}
}
