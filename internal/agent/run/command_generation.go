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

func (s *execution) generateObserved(ctx context.Context, request modelapi.Request, consume func(modelapi.Event) error) error {
	commands, ok := s.runner.Tools.(toolapi.CommandExecutor)
	if !ok || len(commands.CommandActivity()) == 0 {
		return s.runner.Generator.Generate(ctx, request, consume)
	}
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	type envelope struct {
		event  modelapi.Event
		result chan error
	}
	events := make(chan envelope)
	done := make(chan error, 1)
	go func() {
		done <- s.runner.Generator.Generate(child, request, func(event modelapi.Event) error {
			reply := make(chan error, 1)
			select {
			case events <- envelope{event, reply}:
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
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var failure error
	for {
		select {
		case message := <-events:
			if failure == nil {
				_, failure = s.collectCommands(ctx)
				if failure == nil {
					failure = consume(message.event)
				}
			}
			message.result <- failure
			if failure != nil {
				cancel()
			}
		case <-ticker.C:
			if failure == nil {
				_, failure = s.collectCommands(ctx)
				if failure != nil {
					cancel()
				}
			}
		case err := <-done:
			return errors.Join(err, failure)
		case <-ctx.Done():
			cancel()
			return errors.Join(ctx.Err(), <-done, failure)
		}
	}
}
