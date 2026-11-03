// Purpose: Verify request-local refusals follow the frozen operation across peer names.
// Pattern: Concurrent policy identity regression tests.
// Usage: go test -race ./internal/tools/team/approval.
// Related: internal/tools/team/approval/ledger.go, internal/commands/teamrequest/approval.go.
package approval

import (
	"sync"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestRefusalTracksCandidateAcrossPeersAndStaysRequestLocal(t *testing.T) {
	var ledger, other Ledger
	proposal := toolapi.Approval{Name: "Hana · write_file", Candidate: &toolapi.Candidate{Binding: "frozen-operation"}}
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() { ledger.Deny(proposal) })
	}
	workers.Wait()
	proposal.Name = "Ren · write_file"
	if !ledger.Refused(proposal) || other.Refused(proposal) {
		t.Fatal("refusal was bypassed by renaming or escaped its request")
	}
	proposal.Candidate = &toolapi.Candidate{Binding: "different-operation"}
	if ledger.Refused(proposal) {
		t.Fatal("refusal blocked a different candidate")
	}
}
