// Purpose: Provide controllable receipt admission without creating a server-to-adapter import cycle.
// Pattern: In-memory persistence test double.
// Usage: Server mutation tests exercise cancellation and exactly-once effects.
// Related: internal/commands/server/file_mutation_test.go, internal/commands/server/file_mutation_receipt.go.
package server

import (
	"context"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"sync"
)

type mutationLedgerFixture struct {
	mu      sync.Mutex
	values  map[string]FileMutationResult
	digests map[string][32]byte
}

func (l *mutationLedgerFixture) Admit(ctx context.Context, session, key string, digest [32]byte) (FileMutationResult, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	id := session + "/" + key
	if value, ok := l.values[id]; ok {
		if l.digests[id] != digest {
			return FileMutationResult{}, false, fault.New(diag.Conflict, diag.FileMutationConflict)
		}
		return value, true, nil
	}
	if l.digests == nil {
		l.digests = make(map[string][32]byte)
	}
	l.digests[id] = digest
	value := FileMutationResult{Key: key, State: "pending"}
	l.values[id] = value
	return value, false, ctx.Err()
}
func (l *mutationLedgerFixture) Lookup(ctx context.Context, session, key string) (FileMutationResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.values[session+"/"+key], ctx.Err()
}
func (l *mutationLedgerFixture) Finish(ctx context.Context, session, key string, value FileMutationResult) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.values[session+"/"+key] = value
	return ctx.Err()
}

type pausedMutationLedger struct {
	*mutationLedgerFixture
	admitted, release chan struct{}
}

func (l *pausedMutationLedger) Admit(ctx context.Context, session, key string, digest [32]byte) (FileMutationResult, bool, error) {
	value, duplicate, err := l.mutationLedgerFixture.Admit(ctx, session, key, digest)
	if err == nil && !duplicate {
		close(l.admitted)
		select {
		case <-l.release:
		case <-ctx.Done():
			return value, false, ctx.Err()
		}
	}
	return value, duplicate, err
}
