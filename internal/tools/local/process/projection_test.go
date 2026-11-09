// Purpose: Verify output-memory admission across independent registries and release on projection failures.
// Pattern: Controlled concurrent budget and collector regression tests.
// Usage: go test -race ./internal/tools/local/process -run TestProjection.
// Related: internal/tools/local/process/projection.go, internal/tools/local/process/projection_read.go.
package process

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestProjectionBudgetIsSharedAndDoesNotAllocateForSmallOutput(t *testing.T) {
	ctx := WithProjectionBudget(t.Context())
	var a, b Registry
	a.Begin(ctx, time.Now().Add(time.Minute))
	defer a.End()
	b.Begin(ctx, time.Now().Add(time.Minute))
	defer b.End()
	if a.ProjectionBudget() != b.ProjectionBudget() {
		t.Fatal("team registries did not share output admission")
	}
	budget := a.ProjectionBudget()
	if allocs := testing.AllocsPerRun(100, func() {
		release, err := budget.Acquire(t.Context(), 1024)
		if err != nil {
			t.Error(err)
			return
		}
		release()
	}); allocs != 0 {
		t.Fatal("small output allocated admission state", allocs)
	}
}
func TestProjectionBudgetSerializesOnlyLargeMaterializations(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var budget ProjectionBudget
		release, err := budget.Acquire(t.Context(), 32<<20)
		if err != nil {
			t.Fatal(err)
		}
		admitted, done := make(chan struct{}), make(chan struct{})
		go func() {
			release, err := budget.Acquire(t.Context(), 1<<20)
			close(admitted)
			if err != nil {
				t.Error(err)
				return
			}
			release()
			close(done)
		}()
		synctest.Wait()
		select {
		case <-admitted:
			t.Fatal("large projections exceeded the shared memory budget")
		default:
		}
		release()
		<-done
		if budget.used != 0 {
			t.Fatal("projection permit leaked", budget.used)
		}
	})
}
