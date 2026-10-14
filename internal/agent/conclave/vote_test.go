// Purpose: Verify Conclave scaling, quorum and strict plain-text ballots.
// Pattern: Review policy tests.
// Usage: go test ./internal/agent/conclave.
// Related: internal/agent/conclave/vote.go, internal/agent/conclave/review.go.
package conclave

import "testing"

func TestReviewCounts(t *testing.T) {
	for _, tc := range []struct{ lines, n, need int }{{0, 0, 0}, {1, 3, 2}, {75, 3, 2}, {76, 6, 4}, {80, 6, 4}, {150, 6, 4}, {151, 9, 6}, {225, 9, 6}, {226, 12, 8}, {10000, 12, 8}} {
		n, need := Allocation(tc.lines)
		if n != tc.n || need != tc.need {
			t.Fatalf("L=%d got %d/%d", tc.lines, n, need)
		}
	}
}
func TestBallotGrammar(t *testing.T) {
	if vote, err := ParseVote("1"); err != nil || !vote.Agree {
		t.Fatalf("vote=%+v %v", vote, err)
	}
	if vote, err := ParseVote("0\nThe proposed edit does not implement the task."); err != nil || vote.Agree || vote.Reason == "" {
		t.Fatalf("vote=%+v %v", vote, err)
	}
	for _, text := range []string{"0", "yes", "1\nmaybe", `{"vote":1}`} {
		if _, err := ParseVote(text); err == nil {
			t.Fatalf("invalid vote accepted: %s", text)
		}
	}
}
