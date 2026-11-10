// Purpose: Resolve uncertain client submissions without repeating model or file effects.
// Pattern: Read-only durable request lookup.
// Usage: Browser retry reconciliation queries the original session/key pair.
// Related: internal/commands/server/identity.go, internal/commands/server/tasks.go.
package server

import (
	"context"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"strings"
	"unicode/utf8"
)

func (s *Service) LookupRequest(ctx context.Context, session, key string) (Task, error) {
	if !sessionstate.ValidID(session) || len(key) < 8 || len(key) > 128 || !utf8.ValidString(key) || strings.ContainsAny(key, "\r\n\x00") {
		return Task{}, fault.New(diag.InvalidArgument, diag.ServerPageInvalid)
	}
	return s.TaskStatus(ctx, TaskID(session, key))
}
