// Purpose: Preserve completed mutations when cancellation or presentation failure arrives at the result boundary.
// Pattern: Execution boundary regressions.
// Usage: go test ./internal/agent/run -run TestAppliedEffect.
// Related: internal/agent/run/action_result.go, internal/agent/run/execution_result.go.
package run

import (
	"context"
	"errors"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/work"
)

func TestAppliedEffectSurvivesResultDeliveryFailure(t *testing.T) {
	for _, interrupt := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		r, _, _, _, _ := realBatchRunner(t, nil)
		r.Generator = &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "written"})}}
		sentinel := errors.New("display failed")
		result, err := r.Run(ctx, "write a", approve, func(e runrecord.Event) error {
			if e.Kind == "tool_succeeded" {
				if interrupt {
					cancel()
					return ctx.Err()
				}
				return sentinel
			}
			return nil
		})
		cancel()
		if err == nil || len(result.Facts.Files) != 1 || !result.Facts.Files[0].Changed {
			t.Fatalf("lost applied effect: %+v %v", result, err)
		}
		if interrupt && result.End.Status != work.Interrupted || !interrupt && !errors.Is(err, sentinel) {
			t.Fatalf("lost failure kind: %+v %v", result, err)
		}
	}
}
