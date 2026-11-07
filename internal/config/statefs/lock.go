// Purpose: Acquire an exclusive directory lease without losing request cancellation.
// Pattern: Cancellable process-shared filesystem lock.
// Usage: Configuration transactions and durable storage reservations serialize updates here.
// Related: internal/config/statefs/lease_unix.go, internal/config/update.go.
package statefs

import (
	"context"
	"errors"
	"os"
	"syscall"
	"time"
)

func Lock(ctx context.Context, root *os.Root) (*os.File, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		lease, err := Lease(root, true)
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			return lease, err
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
