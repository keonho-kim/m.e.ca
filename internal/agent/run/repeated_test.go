// Purpose: Verify applied mutations are reused without suppressing necessary batch siblings.
// Pattern: Filesystem-backed runtime regressions.
// Usage: go test ./internal/agent/run -run TestAppliedMutation.
// Related: internal/agent/run/repeated.go, internal/agent/run/batch_wave.go.
package run

import (
	"context"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"os"
	"path/filepath"
	"testing"
)

func TestAppliedMutationReuseKeepsPendingSibling(t *testing.T) {
	r, _, root, j, _ := realBatchRunner(t, nil)
	first := nativeCall("write_file", map[string]any{"path": "a", "content": "first"})
	calls := []toolapi.Call{batchCall("reuse", "write_file", map[string]any{"path": "a", "content": "first"}), batchCall("new", "write_file", map[string]any{"path": "b", "content": "second"})}
	g := &generator{replies: []string{first, "batch placeholder", "Both requested files exist."}}
	r.Generator = &reuseGenerator{script: g, calls: calls}
	result, err := r.Run(context.Background(), "Create a and b", approve, emit)
	if err != nil || result.Status != Answered {
		t.Fatalf("%+v %v", result, err)
	}
	for _, name := range []string{"a", "b"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	started, reused := 0, 0
	for _, event := range j.events {
		if event.Kind == "tool_started" {
			started++
		}
		if event.Kind == "tool_reused" {
			reused++
		}
	}
	if started != 2 || reused != 1 {
		t.Fatalf("started=%d reused=%d", started, reused)
	}
}

type reuseGenerator struct {
	script *generator
	calls  []toolapi.Call
}

func (g *reuseGenerator) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	return g.script.Generate(ctx, request, func(event modelapi.Event) error {
		if len(request.Tools) > 0 && len(g.script.requests) == 2 {
			if event.Done {
				return emit(modelapi.Event{Calls: g.calls, Done: true})
			}
			return nil
		}
		return emit(event)
	})
}
