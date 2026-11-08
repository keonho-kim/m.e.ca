// Purpose: Preserve an explicit failed observation when command output cannot enter finalization.
// Pattern: Fake-clock memory saturation regression test.
// Usage: go test ./internal/tools/local/process -run TestFinalizationAdmissionFailure.
// Related: internal/tools/local/process/command_finish.go, internal/tools/local/process/projection.go.
package process

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func TestFinalizationAdmissionFailureRecordsUncertainCapture(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		budget := &ProjectionBudget{}
		release, err := budget.Acquire(t.Context(), projectionBytes)
		if err != nil {
			t.Fatal(err)
		}
		defer release()
		r := &Registry{projection: budget}
		j := &job{}
		if _, err := j.stdout.Write(make([]byte, 128<<10)); err != nil {
			t.Fatal(err)
		}
		saves := 0
		r.finish(context.Background(), "cmd_owned", j, startedCommand{wait: func() error { return nil }, output: func(error) (Output, error) { t.Fatal("unadmitted full capture was materialized"); return Output{}, nil }}, func(_ string, _ string, output Output, err error) error {
			saves++
			if !output.Started || !output.Truncated || output.Status != catalog.ExecutionFailed || fault.Code(err) != catalog.PersistenceFailed {
				t.Fatal("failed capture was reported as successful execution", output, err)
			}
			return nil
		})
		if saves != 1 || j.state.Error == "" || j.stdout.head.Len() != 0 {
			t.Fatal("missing failure record or retained capture", saves, j.state)
		}
	})
}
