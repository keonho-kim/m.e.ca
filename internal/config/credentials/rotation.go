// Purpose: Maintain stored encryption keys periodically without a user-facing command or setting.
// Pattern: Process-owned cancellable worker with persistent per-version scheduling.
// Usage: cmd/meca starts one worker after acquiring state leases and joins it before releasing them.
// Related: internal/storage/credentialstore/rotation.go, cmd/meca/main.go.
package credentials

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/keonho-kim/meca/internal/config/statefs"
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
	var diagnosticErr error
	for {
		if ctx.Err() != nil {
			return diagnosticErr
		}
		at := now()
		err := v.rotateOnce(ctx, at)
		if ctx.Err() != nil {
			return diagnosticErr
		}
		if err != nil {
			// Backend errors can contain untrusted text. Persist only a fixed
			// bounded diagnostic, never key values or raw OS/SQL error output.
			body := at.UTC().Format(time.RFC3339) + " credential key rotation deferred; current API credentials retained; retry scheduled\n"
			diagnosticErr = nil
			if err := statefs.Write(filepath.Join(v.State, "logs"), "credential-rotation.log", []byte(body)); err != nil {
				diagnosticErr = fmt.Errorf("record credential maintenance failure: %w", err)
			}
		} else {
			diagnosticErr = nil
		}
		select {
		case <-ctx.Done():
			return diagnosticErr
		case _, ok := <-ticks:
			if !ok {
				return diagnosticErr
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
		return fmt.Errorf("credential state directory is required")
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
