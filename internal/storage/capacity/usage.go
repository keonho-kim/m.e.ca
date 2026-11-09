// Purpose: Read conservative persistent allocation usage under the counter's process-shared lock.
// Pattern: Read-only capacity query.
// Usage: Server capability discovery reports quota pressure before admission fails.
// Related: internal/storage/capacity/reserve.go, internal/storage/serverrequests/page.go.
package capacity

import (
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/config/statefs"
	"os"
)

func Allocated(ctx context.Context, directory string) (int64, error) {
	root, err := statefs.Open(directory)
	if err != nil {
		return 0, err
	}
	defer root.Close()
	lease, err := statefs.Lock(ctx, root)
	if err != nil {
		return 0, err
	}
	defer lease.Close()
	count, err := readCount(root)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	return count, err
}
