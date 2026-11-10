// Purpose: Reject new admissions at capacity while preserving retry and completion semantics.
// Pattern: Durable quota boundary regression test.
// Usage: go test ./internal/storage/serverrequests -run TestAdmissionCapacity.
// Related: internal/storage/serverrequests/capacity.go, internal/storage/serverrequests/ledger.go.
package serverrequests

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/config/statefs"
	"github.com/keonho-kim/meca/internal/storage/capacity"
)

func TestAdmissionCapacityRetainsDuplicateAndCompletion(t *testing.T) {
	directory := t.TempDir()
	l, err := Open(context.Background(), directory, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	const session = "session_AAAAAAAAAAAAAAAAAAAAAAAAAA"
	first, _, err := l.Admit(t.Context(), session, "original", server.InputDigest("text", nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := statefs.Write(filepath.Join(directory, "capacity"), "allocated", []byte("v1\n"+strconv.Itoa(maxAdmissions)+"\n")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := l.Admit(t.Context(), session, "new", server.InputDigest("text", nil)); !errors.Is(err, capacity.ErrLimit) {
		t.Fatal(err)
	}
	if _, duplicate, err := l.Admit(t.Context(), session, "original", server.InputDigest("text", nil)); err != nil || !duplicate {
		t.Fatal("retry was blocked", err)
	}
	first.Status = "answered"
	if err := l.Finish(t.Context(), first); err != nil {
		t.Fatal("completion was blocked", err)
	}
}
