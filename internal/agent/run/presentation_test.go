// Purpose: Verify stable presentation identity across generation persistence and rejected calls.
// Pattern: Scripted runtime contract tests.
// Usage: go test ./internal/agent/run -run TestPresentation.
// Related: internal/agent/run/presentation.go, internal/agent/run/tool_reply.go.
package run

import (
	"github.com/keonho-kim/meca/internal/agent/record"
	"testing"
)

func TestPresentationRejectedCandidates(t *testing.T) {
	g := &generator{replies: []string{nativeCall("unknown_fixture", map[string]any{}), nativeCall("unknown_fixture", map[string]any{}), "Finished."}}
	r := fixture(g, &executor{}, &journal{})
	r.Generator = repeatedIDs{g}
	var positions, failed []string
	_, err := r.Run(t.Context(), "Inspect.", approve, func(e record.Event) error {
		if e.Kind == "generation_started" {
			positions = append(positions, e.Presentation)
		}
		if e.Kind == "tool_failed" {
			if e.Call == nil || e.Call.ExecutionID == "" || e.Presentation != e.Call.ExecutionID {
				t.Fatalf("missing failure identity: %+v", e)
			}
			failed = append(failed, e.Presentation)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(positions) != 3 || len(failed) != 2 || !(positions[0] < failed[0] && failed[0] < positions[1] && positions[1] < failed[1] && failed[1] < positions[2]) {
		t.Fatal(positions, failed)
	}
}
