// Purpose: Bound durable allocations while preserving all existing user records and backups.
// Pattern: Process-shared reservation counter with explicit publication ownership.
// Usage: Session, request and encryption-key stores reserve before publishing new persistent state.
// Related: internal/config/statefs/lock.go, internal/storage/serverrequests/capacity.go.
package capacity

import (
	"context"
	"errors"

	"io"
	"os"
	"strconv"
	"strings"

	"github.com/keonho-kim/meca/internal/config/statefs"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var ErrLimit = faults.Errorf(diagcatalog.CapacityStorageCapacityLimitReachedExistingRecordsAnd)

// Reserve commits the slot before its side effect. Uncertain or interrupted
// publication keeps the slot, so retries cannot silently exceed the ceiling.
func Reserve(ctx context.Context, directory string, maximum int64, initial func() (int64, error)) (reservation *Reservation, resultErr error) {
	if maximum < 1 {
		return nil, faults.Errorf(diagcatalog.CapacityInvalidStorageAllocationLimit)
	}
	root, err := statefs.Open(directory)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	lease, err := statefs.Lock(ctx, root)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, lease.Close()) }()
	count, err := readCount(root)
	if errors.Is(err, os.ErrNotExist) {
		count, err = initial()
	}
	if err != nil {
		return nil, err
	}
	if count < 0 || count >= maximum {
		diagnostics.Log(ctx, diagcatalog.StorageCapacityLow, "component", "storage", "count", count, "limit", maximum, "status", "full")
		return nil, ErrLimit
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := CheckFree(directory, 1<<30); err != nil {
		return nil, err
	}
	if err := statefs.Write(directory, "allocated", []byte("v1\n"+strconv.FormatInt(count+1, 10)+"\n")); err != nil {
		return nil, err
	}
	if count+1 >= maximum-maximum/10 {
		diagnostics.Log(ctx, diagcatalog.StorageCapacityLow, "component", "storage", "count", count+1, "limit", maximum, "status", "near_limit")
	}
	return &Reservation{directory: directory}, nil
}

func readCount(root *os.Root) (count int64, resultErr error) {
	info, err := root.Lstat("allocated")
	if err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 32 {
		return 0, faults.Errorf(diagcatalog.CapacityInvalidPrivateAllocationCounterContentsPreserved)
	}
	file, err := root.Open("allocated")
	if err != nil {
		return 0, err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	current, err := file.Stat()
	if err != nil {
		return 0, err
	}
	if !os.SameFile(info, current) {
		return 0, faults.Errorf(diagcatalog.CapacityAllocationCounterChangedWhileOpening)
	}
	body, err := io.ReadAll(io.LimitReader(file, 33))
	if err != nil {
		return 0, err
	}
	parts := strings.Split(string(body), "\n")
	if len(body) > 32 || len(parts) != 3 || parts[0] != "v1" || parts[2] != "" {
		return 0, faults.Errorf(diagcatalog.CapacityUnsupportedAllocationCounterContentsPreserved)
	}
	count, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil || count < 0 {
		return 0, faults.Errorf(diagcatalog.CapacityInvalidAllocationCountContentsPreserved)
	}
	return count, nil
}
