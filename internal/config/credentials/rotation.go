// Purpose: Maintain stored encryption keys periodically without a user-facing command or setting.
// Pattern: Process-owned cancellable worker with persistent per-version scheduling.
// Usage: cmd/meca starts one worker after acquiring state leases and joins it before releasing them.
// Related: internal/storage/credentialstore/rotation.go, cmd/meca/main.go.
package credentials

import (
	"context"
	"errors"

	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/credentialstore"
)

const (
	rotationPeriod = 24 * time.Hour
	rotationPoll   = time.Hour
)

// StartRotation returns an idempotent stop function that cancels and joins all
// maintenance I/O. Normal rotation failures leave model use intact and retry.
func (v *Vault) StartRotation(parent context.Context) func() error {
	ctx, cancel := context.WithCancel(parent)
	done := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(rotationPoll)
		defer ticker.Stop()
		done <- v.runRotation(ctx, time.Now, ticker.C)
	}()
	var once sync.Once
	var result error
	return func() error {
		once.Do(func() {
			cancel()
			result = <-done
		})
		return result
	}
}

func (v *Vault) runRotation(ctx context.Context, now func() time.Time, ticks <-chan time.Time) error {
	for {
		if ctx.Err() != nil {
			return nil
		}
		at := now()
		err := faults.Guard(func() error { return v.rotateOnce(ctx, at) })
		if faults.IsPanic(err) {
			return err
		}
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			diagnostics.Log(ctx, diagcatalog.RotationDeferred, "component", "credentials", "error_code", diagcatalog.RotationDeferredCode)
		}
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-ticks:
			if !ok {
				return nil
			}
		}
	}
}

func (v *Vault) rotateOnce(ctx context.Context, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if v.State == "" {
		return faults.Errorf(diagcatalog.CredentialsCredentialStateDirectoryIsRequired)
	}
	// A process without saved credentials needs no database or OS key access.
	if _, err := os.Lstat(filepath.Join(v.State, credentialstore.Filename)); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	store, err := credentialstore.Open(ctx, v.State, v.keyStore(), false)
	if err != nil {
		return err
	}
	return errors.Join(store.RotateDue(ctx, now, rotationPeriod), store.Close())
}
