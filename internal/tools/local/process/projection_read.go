// Purpose: Size live output under collector locks before materializing a budgeted snapshot.
// Pattern: Optimistic metadata observation followed by checked memory admission.
// Usage: Local running-command views release the returned permit after projection and log writes.
// Related: internal/tools/local/process/projection.go, internal/tools/local/process/output.go.
package process

import (
	"errors"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var ErrProjectionBusy = faults.Errorf(diagcatalog.ProcessCommandOutputIsBeingFinalizedInspectThe)

func (r *Registry) ReadProjection(id string, after [2]uint64, positions [2]int64) ([2]Capture, func(), error) {
	j, err := r.lookup(id)
	if err != nil {
		return [2]Capture{}, nil, err
	}
	budget := r.ProjectionBudget()
	var required int64
	for {
		release, available := budget.TryAcquire(required)
		if !available {
			return [2]Capture{}, nil, ErrProjectionBusy
		}
		j.stdout.mu.Lock()
		j.stderr.mu.Lock()
		var actual int64
		if j.stdout.revision != after[0] {
			actual += j.stdout.changedBytesLocked(positions[0])
		}
		if j.stderr.revision != after[1] {
			actual += j.stderr.changedBytesLocked(positions[1])
		}
		if actual > required {
			j.stderr.mu.Unlock()
			j.stdout.mu.Unlock()
			release()
			required = max(actual, required*2)
			continue
		}
		captures := [2]Capture{j.stdout.changesLocked(after[0], positions[0]), j.stderr.changesLocked(after[1], positions[1])}
		err = errors.Join(j.stdout.failure, j.stderr.failure)
		j.stderr.mu.Unlock()
		j.stdout.mu.Unlock()
		return captures, release, err
	}
}
