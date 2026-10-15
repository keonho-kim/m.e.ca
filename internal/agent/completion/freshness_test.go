// Purpose: Verify actual versions restore only matching file facts and retain changed or unknown status.
// Pattern: Pure state reconciliation regression tests.
// Usage: go test ./internal/agent/completion -run TestFreshness.
// Related: internal/agent/completion/freshness.go, internal/agent/completion/state.go.
package completion

import "testing"

func TestFreshnessReconcilesOnlyMatchingKnownFiles(t *testing.T) {
	s := State{Receipts: []Receipt{
		{ID: "unchanged", Tool: "write_file", Kind: "content", Success: true, Stale: true, Versions: map[string]string{"/p/a": "v1"}},
		{ID: "changed", Tool: "read_file", Kind: "content", Success: true, Stale: true, Versions: map[string]string{"/p/b": "v1"}},
		{ID: "unknown", Tool: "read_file", Kind: "content", Success: true, Stale: true, Versions: map[string]string{"/p/c": "v1"}},
		{ID: "command", Tool: "execute_command", Kind: "command", Success: true, Stale: true},
	}}
	s.ReconcileFiles(map[string]string{"/p/a": "v1", "/p/b": "v2"})
	if s.Receipts[0].Stale || !s.Receipts[1].Stale || !s.Receipts[2].Stale || !s.Receipts[3].Stale {
		t.Fatalf("incorrect freshness: %+v", s.Receipts)
	}
}

func TestFreshnessPreservesHistoricalCommandOutcome(t *testing.T) {
	var s State
	s.Observe(Receipt{ID: "test", Kind: "command", Tool: "execute_command", Command: "test command", Success: true, Output: "exit 0"}, nil, true)
	s.Observe(Receipt{ID: "ls", Kind: "command", Tool: "execute_command", Command: "ls", Success: true}, nil, true)
	if s.Receipts[0].Stale || !s.Receipts[0].Success {
		t.Fatal("a later command erased the earlier actual execution result")
	}
}
