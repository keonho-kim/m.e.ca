// Purpose: Preserve encrypted records across failed rotation publication and malformed timing state.
// Pattern: Deterministic storage and key-backend fault tests.
// Usage: go test ./internal/storage/credentialstore -run TestPeriodicRotation.
// Related: internal/storage/credentialstore/rotation.go, internal/storage/credentialstore/rotation_schedule.go.
package credentialstore

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type verificationFailure struct {
	*testKeys
	original string
}

func (k verificationFailure) Get(ctx context.Context, ref string) (string, error) {
	if ref != k.original {
		return "", errors.New("readback unavailable")
	}
	return k.testKeys.Get(ctx, ref)
}

func TestPeriodicRotationPublicationFailuresPreserveValue(t *testing.T) {
	for _, stage := range []string{"save", "readback", "sql"} {
		t.Run(stage, func(t *testing.T) {
			s, keys, _ := credentialFixture(t)
			if err := s.Put(t.Context(), "provider", "scope", "unchanged-secret"); err != nil {
				t.Fatal(err)
			}
			now := time.Unix(1_800_000_000, 0)
			if err := s.RotateDue(t.Context(), now, time.Hour); err != nil {
				t.Fatal(err)
			}
			old, sealed := currentCipher(t, s, "provider")
			switch stage {
			case "save":
				keys.failSet = true
			case "readback":
				s.keys = verificationFailure{testKeys: keys, original: old}
			case "sql":
				if _, err := s.db.Exec(`CREATE TRIGGER reject_rotation BEFORE UPDATE ON credentials BEGIN SELECT RAISE(ABORT,'write rejected'); END`); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.RotateDue(t.Context(), now.Add(time.Hour), time.Hour); err == nil {
				t.Fatal("failed publication was reported as success")
			}
			if got, raw := currentCipher(t, s, "provider"); got != old || !bytes.Equal(raw, sealed) {
				t.Fatal("failed publication changed the original credential")
			}
			keys.failSet, s.keys = false, keys
			if got, err := s.Get(t.Context(), "provider", "scope"); err != nil || got != "unchanged-secret" {
				t.Fatal("failed rotation lost the API key", err)
			}
		})
	}
}

func TestPeriodicRotationNewVersionAndBackwardsClock(t *testing.T) {
	s, _, _ := credentialFixture(t)
	if err := s.Put(t.Context(), "provider", "scope", "old-secret"); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	if err := s.RotateDue(t.Context(), now, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(t.Context(), "provider", "scope", "new-secret"); err != nil {
		t.Fatal(err)
	}
	saved, _ := currentCipher(t, s, "provider")
	if err := s.RotateDue(t.Context(), now.Add(2*time.Hour), time.Hour); err != nil {
		t.Fatal(err)
	}
	if got, _ := currentCipher(t, s, "provider"); got != saved {
		t.Fatal("user's new encryption version inherited an old expiry")
	}
	if err := s.RotateDue(t.Context(), now.Add(time.Hour), time.Hour); err != nil {
		t.Fatal(err)
	}
	if got, _ := currentCipher(t, s, "provider"); got == saved {
		t.Fatal("backwards clock postponed key rotation")
	}
}

func TestPeriodicRotationRejectsMalformedOrLinkedSchedule(t *testing.T) {
	for _, linked := range []bool{false, true} {
		s, keys, directory := credentialFixture(t)
		if err := s.Put(t.Context(), "provider", "scope", "secret"); err != nil {
			t.Fatal(err)
		}
		now := time.Unix(1_800_000_000, 0)
		if err := s.RotateDue(t.Context(), now, time.Hour); err != nil {
			t.Fatal(err)
		}
		old, sealed := currentCipher(t, s, "provider")
		name := filepath.Join(directory, rotationDirectory, rotationName("provider"))
		if err := os.Remove(name); err != nil {
			t.Fatal(err)
		}
		if linked {
			if err := os.Symlink(rotationName("missing"), name); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(name, []byte("unsupported-schedule"), 0600); err != nil {
			t.Fatal(err)
		}
		writes := keys.writes
		if err := s.RotateDue(t.Context(), now.Add(time.Hour), time.Hour); err == nil {
			t.Fatal("malformed schedule silently replaced")
		}
		if got, raw := currentCipher(t, s, "provider"); got != old || !bytes.Equal(raw, sealed) || writes != keys.writes {
			t.Fatal("invalid schedule changed encryption material")
		}
	}
}
