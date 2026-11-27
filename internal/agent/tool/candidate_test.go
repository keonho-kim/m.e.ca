// Purpose: Reject changes to the identity or arguments of an approved candidate.
// Pattern: Observable trust-boundary contract tests.
// Usage: go test ./internal/agent/tool -run TestCandidateRejectsChangedCall.
// Related: internal/agent/tool/candidate.go, internal/agent/run/prepare_execution.go.
package tool

import "testing"

func TestCandidateRejectsChangedCall(t *testing.T) {
	call := Call{ID: "write", Name: "write_file", Arguments: []byte(`{"path":"a","content":"approved"}`)}
	candidate := BindCandidate(call, []FileDelta{{Path: "a", Added: 1, Diff: "+approved"}})
	if err := candidate.Check(call); err != nil {
		t.Fatal(err)
	}
	for _, changed := range []Call{
		{ID: "another-call", Name: call.Name, Arguments: call.Arguments},
		{ID: call.ID, Name: "edit_file", Arguments: call.Arguments},
		{ID: call.ID, Name: call.Name, Arguments: []byte(`{"path":"b","content":"approved"}`)},
		{ID: call.ID, Name: call.Name, Arguments: []byte(`{"path":"a","content":"different"}`)},
	} {
		if err := candidate.Check(changed); err == nil {
			t.Fatalf("accepted changed call: %+v", changed)
		}
	}
}
