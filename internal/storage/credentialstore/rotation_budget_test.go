// Purpose: Ensure a slow credential cannot starve subsequent keys across bounded maintenance passes.
// Pattern: Fake-clock persistence and cancellation test.
// Usage: go test -race ./internal/storage/credentialstore -run TestPeriodicRotationResumes.
// Related: internal/storage/credentialstore/rotation_cursor.go, internal/storage/credentialstore/rotation.go.
package credentialstore

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

func TestPeriodicRotationResumesAfterPassDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, keys, _ := credentialFixture(t)
		for _, ref := range []string{"a", "b"} {
			if err := s.Put(t.Context(), ref, "scope", ref+"-key"); err != nil {
				t.Fatal(err)
			}
		}
		now := time.Unix(1800000000, 0)
		if err := s.RotateDue(t.Context(), now, time.Hour); err != nil {
			t.Fatal(err)
		}
		a, _ := currentCipher(t, s, "a")
		b, _ := currentCipher(t, s, "b")
		s.keys = &rotationKeys{testKeys: keys, wait: a, started: make(chan struct{}), release: make(chan struct{})}
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		if err := s.RotateDue(ctx, now.Add(time.Hour), time.Hour); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
		if err := s.RotateDue(t.Context(), now.Add(time.Hour), time.Hour); err != nil {
			t.Fatal("next pass retried the slow prefix", err)
		}
		if got, _ := currentCipher(t, s, "b"); got == b {
			t.Fatal("later healthy credential was starved")
		}
		if got, _ := currentCipher(t, s, "a"); got != a {
			t.Fatal("timed-out credential was overwritten")
		}
	})
}
