// Purpose: Keep per-token delivery free of command polling.
// Pattern: Observable call-count regression tests.
// Usage: go test -race ./internal/agent/run -run TestObservation.
// Related: internal/agent/run/command_generation.go, internal/agent/run/commands.go.
package run

import (
	"context"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type notifyingCommands struct {
	*commandFixture
	updates chan struct{}
	polls   int
}

func (n *notifyingCommands) CommandUpdates() <-chan struct{} { return n.updates }
func (n *notifyingCommands) CollectCommands() ([]toolapi.CommandCompletion, error) {
	n.polls++
	return n.commandFixture.CollectCommands()
}

func TestObservationDoesNotPollCommandsPerToken(t *testing.T) {
	ctx := context.Background()
	j := &journal{}
	e := &notifyingCommands{commandFixture: &commandFixture{executor: &executor{}, active: true}, updates: make(chan struct{}, 1)}
	r := fixture(&generator{}, e.executor, j)
	r.Tools = e
	s := execution{runner: r}
	g := statisticsModel(func(_ context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
		for range 1000 {
			if err := send(modelapi.Event{Delta: "x", Streaming: true}); err != nil {
				return err
			}
		}
		return send(modelapi.Event{Done: true})
	})
	if err := s.generateObserved(ctx, g, modelapi.Request{}, func(modelapi.Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if e.polls != 1 {
		t.Fatal("command state was polled for token deltas", e.polls)
	}
}
