// Purpose: Bound aggregate context reservations and release them exactly once.
// Pattern: Process memory ownership contract test.
// Usage: go test -race ./internal/processmemory -run TestContextReservations.
// Related: internal/processmemory/context.go, internal/agent/run/context_memory.go.
package processmemory

import (
	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"testing"
)

func TestContextReservations(t *testing.T) {
	a, b, c := NewContextLease(), NewContextLease(), NewContextLease()
	defer a.Close()
	defer b.Close()
	defer c.Close()
	if err := a.Resize(ContextBytes / 2); err != nil {
		t.Fatal(err)
	}
	if err := b.Resize(ContextBytes / 2); err != nil {
		t.Fatal(err)
	}
	if err := c.Resize(1); fault.Code(err) != catalog.ResourceExhausted {
		t.Fatal(err)
	}
	a.Close()
	a.Close()
	if err := c.Resize(ContextBytes / 2); err != nil {
		t.Fatal("released context was retained", err)
	}
}
