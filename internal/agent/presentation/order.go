// Purpose: Allocate stable display positions shared by concurrent participants in one request.
// Pattern: Request-owned atomic sequence carried through context.
// Usage: Team and runner entry points share Scope; generations and tool candidates call Next once.
// Related: internal/agent/run/start.go, internal/agent/team/room.go.
package presentation

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync/atomic"
)

type contextKey struct{}
type Sequence struct {
	id   string
	next atomic.Uint64
}

func Scope(ctx context.Context) (context.Context, *Sequence) {
	if sequence, ok := ctx.Value(contextKey{}).(*Sequence); ok {
		return ctx, sequence
	}
	sequence := &Sequence{id: rand.Text()}
	return context.WithValue(ctx, contextKey{}, sequence), sequence
}

// Next is allocated at creation, not when a body grows or an operation completes.
func (s *Sequence) Next() string { return fmt.Sprintf("%s.%020d", s.id, s.next.Add(1)) }
