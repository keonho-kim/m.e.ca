// Purpose: Verify selected-read growth shares capacity without waiting or leaking reservations.
// Pattern: Resource ownership contract tests.
// Usage: go test -race ./internal/processmemory.
// Related: internal/processmemory/payload.go, internal/processmemory/transient.go.
package processmemory

import (
	"testing"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func TestPayloadReadLeaseSharesAndReturnsCapacity(t *testing.T) {
	read := NewPayloadLease()
	defer read.Close()
	if err := read.Resize(8); err != nil {
		t.Fatal(err)
	}
	write, err := AcquirePayload(t.Context(), PayloadBytes-8)
	if err != nil {
		t.Fatal(err)
	}
	defer write()
	if err := read.Resize(9); faults.Code(err) != diagcatalog.ResourceExhausted {
		t.Fatal("retained read waited or exceeded the shared budget", err)
	}
	if err := read.Resize(2); err != nil {
		t.Fatal(err)
	}
	released, err := AcquirePayload(t.Context(), 6)
	if err != nil {
		t.Fatal(err)
	}
	released()
	read.Close()
	read.Close()
	if err := read.Resize(1); err == nil {
		t.Fatal("closed reservation was reused")
	}
	write()
	all, err := AcquirePayload(t.Context(), PayloadBytes)
	if err != nil {
		t.Fatal("read or writer retained its charge", err)
	}
	all()
}
