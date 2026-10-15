// Purpose: Verify exact run and review facts survive a summary that omits them.
// Pattern: Evidence preservation test.
// Usage: go test ./internal/agent/compact -run TestReviewFacts.
// Related: internal/agent/compact/ledger.go, internal/agent/compact/context.go.
package compact

import (
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"strings"
	"testing"
)

func TestReviewFactsSurviveSummaryOmission(t *testing.T) {
	ballot := conclave.Result{CallID: "c", CandidateDigest: strings.Repeat("a", 64), Lines: 2, Attempts: 1, Complete: true, Verdict: conclave.Verdict{Decision: "reject", Findings: []conclave.Finding{{Path: "a", Side: "request", Reason: "escape handling is lost"}}}}
	memory := Memory{Revision: 1, Markdown: "summary omits the review"}
	memory.Ledger.merge(Record{Sequence: 1, Run: &RunFact{ID: "r", Objective: "preserve escapes"}})
	memory.Ledger.merge(Record{Sequence: 2, Review: &ballot})
	if err := memory.Check(4096); err != nil {
		t.Fatal(err)
	}
	text, err := memory.Context()
	if err != nil || !strings.Contains(text, "escape handling is lost") || !strings.Contains(text, "preserve escapes") || !strings.Contains(text, ballot.CandidateDigest) {
		t.Fatalf("lost review facts: %s %v", text, err)
	}
}
