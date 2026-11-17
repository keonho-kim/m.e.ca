// Purpose: Verify independent request lifetimes and shared concurrent participant positions.
// Pattern: Concurrent context ownership regression.
// Usage: go test -race ./internal/agent/presentation.
// Related: internal/agent/presentation/order.go.
package presentation

import (
	"context"
	"sync"
	"testing"
)

func TestScopeSharedAcrossParticipants(t *testing.T) {
	ctx, first := Scope(t.Context())
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	_, inherited := Scope(child)
	if first != inherited {
		t.Fatal("child lost request ownership")
	}
	_, other := Scope(t.Context())
	if first.Next() == other.Next() {
		t.Fatal("requests share identities")
	}
	values := make(chan string, 128)
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			for range 16 {
				values <- inherited.Next()
			}
		})
	}
	workers.Wait()
	close(values)
	seen := map[string]bool{}
	for value := range values {
		if seen[value] {
			t.Fatal("duplicate concurrent position")
		}
		seen[value] = true
	}
}
