// Purpose: Apply one settings mutation to the latest configuration under a process-shared lock.
// Pattern: Cancellable filesystem transaction.
// Usage: Terminal and server composition roots supply Update to settings use cases.
// Related: internal/config/config.go, internal/config/statefs/lease_unix.go.
package config

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/config/statefs"
)

func Update(ctx context.Context, directory string, apply func(*Config)) (next Config, resultErr error) {
	if err := ctx.Err(); err != nil {
		return Config{}, err
	}
	root, err := statefs.Open(filepath.Join(directory, "config-write"))
	if err != nil {
		return Config{}, err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	lease, err := statefs.Lock(ctx, root)
	if err != nil {
		return Config{}, err
	}
	defer func() { resultErr = errors.Join(resultErr, lease.Close()) }()
	next, err = Load(directory)
	if err != nil {
		return Config{}, err
	}
	if err := ctx.Err(); err != nil {
		return Config{}, err
	}
	apply(&next)
	if err := ctx.Err(); err != nil {
		return Config{}, err
	}
	return next, Save(directory, next)
}
