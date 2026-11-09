// Purpose: Check that cancellation-independent response persistence is still time-bounded.
// Pattern: Isolated journal-context probe.
// Usage: go test ./internal/agent/run.
// Related: internal/agent/run/generate.go.
package run

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"testing"
	"time"
)

type reviewThinkingGenerator struct{}

func (reviewThinkingGenerator) Generate(ctx context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
	return emit(modelapi.Event{Delta: "done", Thinking: "thinking", Done: true})
}

type reviewDeadlineJournal struct {
	journal
	seen, bounded bool
}

func (j *reviewDeadlineJournal) Commit(ctx context.Context, s string, entries []runrecord.Entry) error {
	for _, e := range entries {
		if e.Event != nil && e.Event.Kind == "thinking" {
			j.seen = true
			_, j.bounded = ctx.Deadline()
		}
	}
	return j.journal.Commit(ctx, s, entries)
}
func TestThinkingFlushDeadline(t *testing.T) {
	j := &reviewDeadlineJournal{}
	r := fixture(&generator{}, &executor{}, &journal{})
	r.Journal = j
	r.Generator = reviewThinkingGenerator{}
	ctx, cancel := context.WithTimeout(t.Context(), time.Hour)
	defer cancel()
	s, release, err := r.start(ctx, "Review", emit, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err := s.askResponse(ctx); err != nil {
		t.Fatal(err)
	}
	if !j.seen {
		t.Fatal("thinking was not recorded")
	}
	if !j.bounded {
		t.Fatal("thinking flush loses the parent deadline without adding a flush deadline")
	}
}
