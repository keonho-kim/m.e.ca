// Purpose: Keep per-token delivery free of command polling and unchanged LAT source queries.
// Pattern: Observable call-count regression tests.
// Usage: go test -race ./internal/agent/run -run TestObservation.
// Related: internal/agent/run/command_generation.go, internal/agent/run/lat_memory.go.
package run

import (
	"context"
	"testing"

	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type countedMemory struct {
	latmemory.Repository
	marks int
}

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

func (r *countedMemory) LATWatermark(ctx context.Context, session string, after int64) (latmemory.Mark, error) {
	r.marks++
	return r.Repository.LATWatermark(ctx, session, after)
}

func TestObservationSkipsUnchangedLATQueriesAndForegroundAdmission(t *testing.T) {
	ctx := context.Background()
	j := &journal{}
	if err := j.AppendMessage(ctx, "s", modelapi.Message{Role: "user", Content: "small request"}); err != nil {
		t.Fatal(err)
	}
	r := fixture(&generator{}, &executor{}, j)
	repo := &countedMemory{Repository: &journalMemory{journal: j}}
	s := execution{runner: r, usage: &requestBudget{}, latObserved: true, latMemory: &reviewMemory{dirty: true, controller: latmemory.Controller{Repository: repo, Session: "s", Redact: r.Redact}}}
	for range 1000 {
		if err := s.maintainLATMemory(ctx, false); err != nil {
			t.Fatal(err)
		}
	}
	if repo.marks != 1 {
		t.Fatal("unchanged records were repeatedly queried", repo.marks)
	}
	s.latMemory.dirty = true
	g := statisticsModel(func(_ context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
		return send(modelapi.Event{Delta: "foreground response", Done: true})
	})
	if err := s.generateObserved(ctx, g, modelapi.Request{}, func(modelapi.Event) error { return nil }); err != nil || repo.marks != 1 {
		t.Fatal("background admission competed with foreground", err)
	}
	if err := s.maintainLATMemory(ctx, false); err != nil || repo.marks != 2 {
		t.Fatal("new records were not inspected at a safe boundary", err)
	}
}
