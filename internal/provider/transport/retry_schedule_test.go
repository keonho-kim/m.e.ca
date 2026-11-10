// Purpose: Verify fixed retry intervals, bounded attempts and cancellation during each wait.
// Pattern: Transport lifecycle tests with virtual time.
// Usage: go test ./internal/provider/transport -run TestAttemptsFixed.
// Related: internal/provider/transport/retry.go, internal/provider/transport/errors.go.
package transport

import (
	"context"
	"errors"
	"slices"
	"syscall"
	"testing"
	"testing/synctest"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func TestAttemptsFixedScheduleAndLimit(t *testing.T) {
	for _, recover := range []bool{false, true} {
		t.Run(map[bool]string{false: "exhausted", true: "fourth-attempt-recovers"}[recover], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				client := New("", "")
				defer client.Close()
				started := time.Now()
				var offsets []time.Duration
				var notices []modelapi.Retry
				err := client.Attempts(t.Context(), func() error {
					offsets = append(offsets, time.Since(started))
					if recover && len(offsets) == 4 {
						return nil
					}
					return syscall.ECONNREFUSED
				}, func(event modelapi.Retry) error { notices = append(notices, event); return nil })
				if want := []time.Duration{0, time.Second, 4 * time.Second, 9 * time.Second}; !slices.Equal(offsets, want) {
					t.Fatalf("attempt offsets=%v, want %v", offsets, want)
				}
				if recover && err != nil || !recover && !errors.Is(err, syscall.ECONNREFUSED) {
					t.Fatalf("unexpected recovery result: %v", err)
				}
				if len(notices) != 3 {
					t.Fatalf("retry notices=%d, want 3", len(notices))
				}
				for i, notice := range notices {
					if notice.Attempt != i+2 || notice.MaxAttempts != 4 {
						t.Fatalf("incorrect retry ordinal: %+v", notice)
					}
				}
			})
		})
	}
}

func TestAttemptsFixedWaitCancellation(t *testing.T) {
	for _, stop := range []struct {
		after time.Duration
		calls int
	}{{500 * time.Millisecond, 1}, {2 * time.Second, 2}, {6 * time.Second, 3}} {
		t.Run(stop.after.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				client := New("", "")
				defer client.Close()
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				timer := time.AfterFunc(stop.after, cancel)
				defer timer.Stop()
				calls := 0
				started := time.Now()
				err := client.Attempts(ctx, func() error { calls++; return syscall.ECONNREFUSED }, nil)
				if !errors.Is(err, context.Canceled) || !errors.Is(err, syscall.ECONNREFUSED) || calls != stop.calls || time.Since(started) != stop.after {
					t.Fatalf("calls=%d elapsed=%v error=%v", calls, time.Since(started), err)
				}
			})
		})
	}
}

func TestAttemptsFixedScheduleKeepsLongerServerWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := New("", "")
		defer client.Close()
		started := time.Now()
		var offsets []time.Duration
		err := client.Attempts(t.Context(), func() error {
			offsets = append(offsets, time.Since(started))
			if len(offsets) == 4 {
				return nil
			}
			return ParseError(429, nil, "2")
		}, nil)
		if want := []time.Duration{0, 2 * time.Second, 5 * time.Second, 10 * time.Second}; err != nil || !slices.Equal(offsets, want) {
			t.Fatalf("attempt offsets=%v, want %v; error=%v", offsets, want, err)
		}
	})
}
