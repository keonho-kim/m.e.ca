// Purpose: Keep distinct tool executions identifiable when a provider reuses call IDs.
// Pattern: Real runtime transitions with a scripted provider adapter.
// Usage: go test ./internal/agent/run -run TestExecutionIdentity.
// Related: internal/agent/run/execution_identity.go, internal/agent/run/execute.go.
package run

import (
	"context"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
)

type repeatedIDs struct{ source *generator }

func (g repeatedIDs) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	return g.source.Generate(ctx, request, func(event modelapi.Event) error {
		for i := range event.Calls {
			event.Calls[i].ID = "reused-provider-id"
		}
		return emit(event)
	})
}
func TestExecutionIdentityAcrossGenerations(t *testing.T) {
	g := &generator{replies: []string{nativeCall("inspect_fixture", map[string]any{"path": "one"}), nativeCall("inspect_fixture", map[string]any{"path": "two"}), "Finished."}}
	r := fixture(g, &executor{}, &journal{})
	r.Generator = repeatedIDs{g}
	var started, finished []string
	_, err := r.Run(t.Context(), "Inspect both files.", approve, func(event record.Event) error {
		if event.Call == nil {
			return nil
		}
		if event.Kind == "tool_started" {
			started = append(started, event.Call.ExecutionID)
		}
		if event.Kind == "tool_succeeded" {
			finished = append(finished, event.Call.ExecutionID)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(started) != 2 || len(finished) != 2 || started[0] == "" || started[0] == started[1] || started[0] != finished[0] || started[1] != finished[1] {
		t.Fatal("execution identity crossed lifecycle or generation boundaries", started, finished)
	}
}
