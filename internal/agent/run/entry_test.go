// Purpose: Verify only valid native calls reach tool preparation.
// Pattern: Runtime trust-boundary test.
// Usage: go test ./internal/agent/run -run TestNativeEntry.
// Related: internal/agent/tool/arguments.go, internal/agent/run/action.go, internal/agent/run/context.go.
package run

import (
	"context"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"testing"
)

func TestNativeEntryRejectsUnknownToolAndExtraArguments(t *testing.T) {
	g := &generator{replies: []string{
		nativeCall("missing_tool", nil),
		nativeCall("inspect_fixture", map[string]any{"path": "a", "unexpected": true}),
		nativeCall("inspect_fixture", map[string]any{"path": "a"}), "Review the file",
	}}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "inspect", approve, emit)
	if err != nil || result.Status != runrecord.Answered || e.prepared != 1 || e.calls[0].Name != "inspect_fixture" {
		t.Fatalf("result=%+v calls=%+v err=%v", result, e.calls, err)
	}
	for _, index := range []int{0, 1, 2, 3} {
		if len(g.requests[index].Tools) == 0 {
			t.Fatal("native definitions missing from request")
		}
	}
	if len(g.requests) != 4 {
		t.Fatal("unexpected separate grounding generation")
	}
}
