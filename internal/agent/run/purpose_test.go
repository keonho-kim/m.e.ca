// Purpose: Verify JSON entry keeps verification intent inside PFC before any execution.
// Pattern: Runtime contract tests.
// Usage: go test ./internal/agent/run -run TestPFCPurpose.
// Related: internal/agent/run/purpose.go, internal/agent/run/action.go, internal/agent/run/observation.go.
package run

import (
	"context"
	"strings"
	"testing"
)

func TestPFCPurposePreservesVerification(t *testing.T) {
	g := &generator{replies: []string{`{"tool":"execute_command"}`, "test-command", "DONE\nVerified"}, purposes: []string{"check"}}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "verify", false, approve, emit)
	if err != nil || result.Status != Verified || e.executed != 1 || len(g.purposeRequests) != 1 {
		t.Fatalf("result=%+v executed=%d err=%v", result, e.executed, err)
	}
	for _, event := range j.events {
		if event.Operation != nil && event.Operation.Check {
			return
		}
	}
	t.Fatal("verification purpose missing from execution facts")
}

func TestPFCPurposeInvalidReplyNeverExecutes(t *testing.T) {
	g := &generator{replies: []string{`{"tool":"read_file"}`, "a", "WAIT\nNeed clarification"}, purposes: []string{"maybe", "maybe", "maybe"}}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "inspect", false, approve, emit)
	if err != nil || result.Status != NeedsInput || e.prepared != 0 || len(g.purposeRequests) != 3 {
		t.Fatalf("result=%+v prepared=%d err=%v", result, e.prepared, err)
	}
	if !strings.Contains(g.purposeRequests[1].Messages[len(g.purposeRequests[1].Messages)-1].Content, "Correction") {
		t.Fatal("missing purpose correction")
	}
}

func TestPFCPurposeCorrectionAndMutationClassification(t *testing.T) {
	g := &generator{replies: []string{`{"tool":"write_file"}`, "a", "new", `{"tool":"execute_command"}`, "test-command", "DONE\nVerified"}, purposes: []string{"unknown", "check"}}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "change and verify", false, approve, emit)
	if err != nil || result.Status != Verified || e.executed != 2 || len(g.purposeRequests) != 2 {
		t.Fatalf("result=%+v executed=%d err=%v", result, e.executed, err)
	}
	for _, event := range j.events {
		if event.Operation != nil && event.Operation.Name == "write_file" && event.Operation.Check {
			t.Fatal("mutation was classified as a check")
		}
	}
}
