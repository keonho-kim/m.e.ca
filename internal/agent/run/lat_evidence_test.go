// Purpose: Verify preparation shares one source hydration and final review reads fresh evidence.
// Pattern: Runtime integration with counted factual-source reads.
// Usage: Run TestLATPreparationSharesEvidence.
// Related: internal/agent/run/lat_prepare.go, internal/agent/lat/memory/evidence.go.
package run

import (
	"context"
	"testing"

	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type evidenceReads struct {
	latmemory.Repository
	reads int
}

func (r *evidenceReads) LATRecords(ctx context.Context, session string, after, through int64, budget int) ([]latmemory.Record, int64, error) {
	r.reads++
	return r.Repository.LATRecords(ctx, session, after, through, budget)
}

func TestLATPreparationSharesEvidenceButFinalReviewReadsAgain(t *testing.T) {
	r, model, _, store := latFixture("Completed answer")
	if err := store.AppendMessage(context.Background(), "s", modelapi.Message{Role: "user", Content: "Retain the earlier requirement"}); err != nil {
		t.Fatal(err)
	}
	repo := &evidenceReads{Repository: r.ReviewMemory}
	r.ReviewMemory, r.RedactionVersion = repo, func() int { return 0 }
	result, err := r.Run(context.Background(), "Implement the request", approve, emit)
	if err != nil || result.Text != "Completed answer" {
		t.Fatal(result, err)
	}
	if repo.reads != 2 {
		t.Fatal("expected one hydration for preparation and a fresh one for final review", repo.reads)
	}
	for _, phase := range []string{"plan", "critique", "refine", "review"} {
		assertContains(t, model.requests[phase][0], "Retain the earlier requirement")
	}
}
