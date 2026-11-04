// Purpose: Keep presence-only execution paths independent of full display snapshots.
// Pattern: Observable query-count regressions.
// Usage: Runtime tests distinguish empty registries and pending completion ownership.
// Related: internal/agent/run/command_generation.go, internal/agent/run/commands.go.
package run

import (
	"context"
	"testing"

	model "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/tool"
)

type countedCommandQueries struct {
	*commandFixture
	snapshots, collections int
}

func (e *countedCommandQueries) CommandActivity() []tool.CommandRun {
	e.snapshots++
	return e.commandFixture.CommandActivity()
}

func (e *countedCommandQueries) CollectCommands() ([]tool.CommandCompletion, error) {
	e.collections++
	return nil, nil
}

func TestGenerationPresenceDoesNotMaterializeCommandLists(t *testing.T) {
	for _, pending := range []bool{false, true} {
		e := &countedCommandQueries{commandFixture: &commandFixture{executor: &executor{}, completed: pending}}
		r := fixture(&generator{}, e.executor, &journal{})
		r.Tools = e
		s := execution{runner: r}
		g := boundaryModel(func(_ context.Context, _ model.Request, send func(model.Event) error) error {
			return send(model.Event{Done: true})
		})
		if err := s.generateObserved(context.Background(), g, model.Request{}, func(model.Event) error { return nil }); err != nil {
			t.Fatal(err)
		}
		if e.snapshots != 0 || pending && e.collections != 1 || !pending && e.collections != 0 {
			t.Fatal("presence lost a completion or materialized display metadata", pending, e.snapshots, e.collections)
		}
	}
}

func TestEmptyFinalCommandCheckAvoidsAFullDisplaySnapshot(t *testing.T) {
	e := &countedCommandQueries{commandFixture: &commandFixture{executor: &executor{}}}
	r := fixture(&generator{}, e.executor, &journal{})
	r.Tools = e
	s := execution{runner: r}
	if pending, err := s.pendingCommands(context.Background()); pending || err != nil || e.snapshots != 0 {
		t.Fatal("empty final check built a display list", pending, e.snapshots, err)
	}
}
