// Purpose: Preserve unsupported request records and isolate project-scoped admissions under one quota.
// Pattern: Durable repository boundary tests.
// Usage: go test ./internal/storage/serverrequests -run 'TestLegacy|TestProject'.
// Related: internal/storage/serverrequests/database.go, internal/storage/serverrequests/capacity.go.
package serverrequests

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/config/statefs"
	"github.com/keonho-kim/meca/internal/storage/capacity"
)

func TestLegacyRequestRecordsRemainIntact(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "resp_original.json")
	const original = `{"version":1,"state":"unresolved"}`
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	l, err := Open(t.Context(), directory, server.ProjectScope("fixture"))
	if l != nil || err == nil {
		if l != nil {
			_ = l.Close()
		}
		t.Fatal("legacy records were silently bypassed", err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != original {
		t.Fatal("legacy records were changed", err)
	}
}

func TestProjectIsolationUsesGlobalCapacity(t *testing.T) {
	directory := t.TempDir()
	a, err := Open(t.Context(), directory, server.ProjectScope("a"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := Open(t.Context(), directory, server.ProjectScope("b"))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	const session = "session_AAAAAAAAAAAAAAAAAAAAAAAAAA"
	first, _, err := a.Admit(t.Context(), session, "same-key", server.InputDigest("a", nil))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Get(t.Context(), first.ID); !errors.Is(err, server.ErrTaskNotFound) {
		t.Fatal("cross-project task became visible", err)
	}
	if _, duplicate, err := b.Admit(t.Context(), session, "same-key", server.InputDigest("b", nil)); err != nil || duplicate {
		t.Fatal("cross-project admission collided", duplicate, err)
	}
	if err := statefs.Write(filepath.Join(directory, "capacity"), "allocated", []byte("v1\n"+strconv.Itoa(maxAdmissions)+"\n")); err != nil {
		t.Fatal(err)
	}
	for _, owner := range []*Ledger{a, b} {
		if _, _, err := owner.Admit(t.Context(), session, "overflow", server.InputDigest("text", nil)); !errors.Is(err, capacity.ErrLimit) {
			t.Fatal("project bypassed the global quota", err)
		}
	}
}

func TestProjectDatabaseRejectsAnotherOwner(t *testing.T) {
	directory := t.TempDir()
	a, err := Open(t.Context(), directory, server.ProjectScope("a"))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(a.path)
	if err != nil {
		t.Fatal(err)
	}
	scope := server.ProjectScope("b")
	folder := filepath.Join(directory, scope)
	if err := os.Mkdir(folder, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(folder, Filename)
	if err := os.WriteFile(path, before, 0600); err != nil {
		t.Fatal(err)
	}
	other, err := Open(t.Context(), directory, scope)
	if other != nil {
		_ = other.Close()
		t.Fatal("project database adopted another owner's records")
	}
	if err == nil {
		t.Fatal("project ownership mismatch was accepted")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatal("foreign project contents were changed", err)
	}
}
