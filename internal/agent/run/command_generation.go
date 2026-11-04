// Purpose: Collect completed commands on the runtime owner while a model request is in flight.
// Pattern: Joined I/O worker with synchronous event handoff.
// Usage: generate uses this boundary when commands may outlive their initial wait.
// Related: internal/agent/run/commands.go, internal/agent/run/usage.go.
package run

import (
	"context"
	"errors"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (s *execution) generateObserved(ctx context.Context, generator modelapi.Generator, request modelapi.Request, consume func(modelapi.Event) error) error {
	commands, ok := s.runner.Tools.(toolapi.CommandExecutor)
	if !ok || !commands.HasCommands() {
		return generator.Generate(ctx, request, consume)
	}
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	// Provider callbacks are serial. Reuse one acknowledgement across events;
	// its reply preserves backpressure and error ordering without per-token channels.
	events := make(chan modelapi.Event)
	reply := make(chan error, 1)
	done := make(chan error, 1)
	go func() {
		done <- generator.Generate(child, request, func(event modelapi.Event) error {
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
		})
	}()
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
				_, failure = s.collectCommands(ctx)
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
