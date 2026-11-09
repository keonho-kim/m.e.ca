// Purpose: Preserve a shared allocation ceiling across concurrency, reopen and malformed state.
// Pattern: Real temporary-filesystem reservation tests.
// Usage: go test -race ./internal/storage/capacity.
// Related: internal/storage/capacity/reserve.go.
package capacity

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestReservationIsDurableAndAtomic(t *testing.T) {
	directory := t.TempDir()
	var accepted atomic.Int32
	var initializations atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			reservation, err := Reserve(t.Context(), directory, 3, func() (int64, error) { initializations.Add(1); return 1, nil })
			if err == nil {
				reservation.Publish()
				if err := reservation.Close(); err != nil {
					t.Error(err)
				}
				accepted.Add(1)
			} else if !errors.Is(err, ErrLimit) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if accepted.Load() != 2 || initializations.Load() != 1 {
		t.Fatal(accepted.Load(), initializations.Load())
	}
	if _, err := Reserve(t.Context(), directory, 3, func() (int64, error) { t.Fatal("counter was recreated"); return 0, nil }); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "allocated"), []byte("unsupported"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Reserve(t.Context(), directory, 3, func() (int64, error) { t.Fatal("malformed counter replaced"); return 0, nil }); err == nil {
		t.Fatal("malformed counter accepted")
	}
}

func TestUnusedReservationIsReturnedExactlyOnce(t *testing.T) {
	directory := t.TempDir()
	reserve := func() *Reservation {
		r, err := Reserve(t.Context(), directory, 2, func() (int64, error) { return 0, nil })
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	unused, published := reserve(), reserve()
	published.Publish()
	if err := published.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Reserve(t.Context(), directory, 2, nil); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if err := unused.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	replacement := reserve()
	replacement.Publish()
	if _, err := Reserve(t.Context(), directory, 2, nil); !errors.Is(err, ErrLimit) {
		t.Fatal("double refund", err)
	}
}
