// Purpose: Verify request-local refusals follow the frozen operation across peer names.
// Pattern: Concurrent policy identity regression tests.
// Usage: go test -race ./internal/tools/team/approval.
// Related: internal/tools/team/approval/ledger.go, internal/commands/teamrequest/approval.go.
package approval

import (
	"strings"
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
	if !ledger.Refused(proposal, nil) || other.Refused(proposal, nil) {
		t.Fatal("refusal was bypassed by renaming or escaped its request")
	}
	proposal.Candidate = &toolapi.Candidate{Binding: "different-operation"}
	if ledger.Refused(proposal, nil) {
		t.Fatal("refusal blocked a different candidate")
	}
}

func TestEmptyRefusalLedgerDoesNotCopyLargeCommand(t *testing.T) {
	var ledger Ledger
	proposal := toolapi.Approval{Command: &toolapi.CommandApproval{Directory: "/project"}, Target: strings.Repeat("command ", 1<<17)}
	if allocs := testing.AllocsPerRun(100, func() {
		if ledger.Refused(proposal, nil) {
			t.Fatal("empty ledger refused command")
		}
	}); allocs != 0 {
		t.Fatal("empty refusal check allocated", allocs)
	}
	ledger.Deny(proposal)
	var key string
	if !ledger.Refused(proposal, &key) {
		t.Fatal("command denial did not survive bounded hashing")
	}
	if allocs := testing.AllocsPerRun(100, func() { ledger.Refused(proposal, &key) }); allocs != 0 {
		t.Fatal("frozen command identity was rehashed", allocs)
	}
}
