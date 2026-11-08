// Purpose: Return an unused durable slot once, while preserving uncertain published allocations.
// Pattern: Request-owned reservation with bounded cleanup.
// Usage: Allocation callers defer Close and call Publish before the first irreversible publication.
// Related: internal/storage/capacity/reserve.go, internal/config/statefs/lock.go.
package capacity

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Reservation struct {
	directory string
	finished  atomic.Bool
}

// Publish keeps the slot even when publication's outcome is uncertain. Call it
// before committing a transaction, linking a session or writing an OS key.
func (r *Reservation) Publish() { r.finished.Store(true) }

// Close returns a slot only if publication never began. A failed counter write
// is not retried: it could already have taken effect. Cleanup outlives request
// cancellation, but cannot hold shutdown for more than five seconds.
func (r *Reservation) Close() (resultErr error) {
	if !r.finished.CompareAndSwap(false, true) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	root, err := statefs.Open(r.directory)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	lease, err := statefs.Lock(ctx, root)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, lease.Close()) }()
	count, err := readCount(root)
	if err != nil {
		return err
	}
	if count < 1 {
		return faults.Errorf(diagcatalog.CapacityInvalidAllocationCountContentsPreserved)
	}
	return statefs.Write(r.directory, "allocated", []byte("v1\n"+strconv.FormatInt(count-1, 10)+"\n"))
}
