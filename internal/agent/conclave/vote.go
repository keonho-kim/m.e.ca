// Purpose: Define line-based review allocation and auditable plain-text voting results.
// Pattern: Pure policy and parser.
// Usage: Parallel reviewers return ballots that Result validates before approval can continue.
// Related: internal/agent/conclave/review.go, internal/agent/run/conclave.go.
package conclave

import (
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"
)

type Vote struct {
	Index  int
	Agree  bool
	Reason string
	Error  string
}
type Result struct {
	CallID, CandidateDigest             string
	Lines, Reviews, Required, Approvals int
	Complete, Passed                    bool
	Votes                               []Vote
}

func Allocation(lines int) (reviews, required int) {
	if lines < 1 {
		return 0, 0
	}
	units := min(4, 1+(lines-1)/75)
	return 3 * units, 2 * units
}
func ParseVote(text string) (Vote, error) {
	text = strings.TrimSpace(text)
	if !utf8.ValidString(text) || len(text) > 2048 {
		return Vote{}, fmt.Errorf("invalid Conclave ballot size or encoding")
	}
	if text == "1" {
		return Vote{Agree: true}, nil
	}
	head, reason, found := strings.Cut(text, "\n")
	if head != "0" || !found || strings.TrimSpace(reason) == "" {
		return Vote{}, fmt.Errorf("Conclave requires 1, or 0 followed by a rejection reason")
	}
	return Vote{Reason: strings.TrimSpace(reason)}, nil
}
func (r Result) Check() error {
	n, required := Allocation(r.Lines)
	digest, digestErr := hex.DecodeString(r.CandidateDigest)
	if r.CallID == "" || len(r.CallID) > 128 || digestErr != nil || len(digest) != 32 || n == 0 || r.Reviews != n || r.Required != required || len(r.Votes) != n {
		return fmt.Errorf("invalid Conclave result binding or allocation")
	}
	approvals := 0
	complete := true
	for i, vote := range r.Votes {
		if vote.Index != i+1 || len(vote.Reason) > 2048 || len(vote.Error) > 2048 || !utf8.ValidString(vote.Reason+vote.Error) {
			return fmt.Errorf("invalid Conclave reviewer record")
		}
		if vote.Error != "" {
			complete = false
			if vote.Agree {
				return fmt.Errorf("failed reviewer cannot approve")
			}
		}
		if vote.Agree {
			approvals++
		} else if vote.Error == "" && strings.TrimSpace(vote.Reason) == "" {
			return fmt.Errorf("rejection has no reason")
		}
	}
	if r.Approvals != approvals || r.Complete != complete || r.Passed != (complete && approvals >= required) {
		return fmt.Errorf("invalid Conclave tally")
	}
	return nil
}
func (r Result) Feedback() string {
	var text strings.Builder
	fmt.Fprintf(&text, "Conclave candidate %s rejected (%d/%d approvals; need %d). No tool execution or user approval occurred.\n", r.CallID, r.Approvals, r.Reviews, r.Required)
	for _, vote := range r.Votes {
		if vote.Error != "" {
			fmt.Fprintf(&text, "- Reviewer %d unavailable: %s\n", vote.Index, vote.Error)
		} else if !vote.Agree {
			fmt.Fprintf(&text, "- Reviewer %d: %s\n", vote.Index, vote.Reason)
		}
	}
	text.WriteString("Reassess these reasons against current evidence, complete a new PFC candidate and self-grounding, then submit it for a fresh review. Do not claim it executed.")
	return text.String()
}
