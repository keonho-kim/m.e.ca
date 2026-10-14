// Purpose: Verify exact Goal and review facts survive a summary that omits them.
// Pattern: Evidence preservation test.
// Usage: go test ./internal/agent/compact -run TestReviewFacts.
// Related: internal/agent/compact/ledger.go, internal/agent/compact/context.go.
package compact

import (
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"github.com/keonho-kim/v-code/internal/agent/goal"
	"strings"
	"testing"
)

func TestReviewFactsSurviveSummaryOmission(t *testing.T) {
	state := goal.State{ID: "g", Request: "fix parser", Objective: "fix parser", Status: "running", Tasks: []goal.Task{{ID: "t", Objective: "preserve escapes", Kind: "work", Status: "running"}}}
	ballot := conclave.Result{CallID: "c", CandidateDigest: strings.Repeat("a", 64), Lines: 2, Reviews: 3, Required: 2, Complete: true, Votes: []conclave.Vote{{Index: 1, Reason: "escape handling is lost"}, {Index: 2, Reason: "no parser test"}, {Index: 3, Reason: "scope mismatch"}}}
	memory := Memory{Revision: 1, Markdown: "summary omits the review"}
	memory.Ledger.merge(Record{Sequence: 1, Goal: &state})
	memory.Ledger.merge(Record{Sequence: 2, Review: &ballot})
	if err := memory.Check(4096); err != nil {
		t.Fatal(err)
	}
	text, err := memory.Context()
	if err != nil || !strings.Contains(text, "escape handling is lost") || !strings.Contains(text, "preserve escapes") || !strings.Contains(text, ballot.CandidateDigest) {
		t.Fatalf("lost review facts: %s %v", text, err)
	}
}
