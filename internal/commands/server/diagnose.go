// Purpose: Explicitly diagnose a connection without saving its model or credentials.
// Pattern: Request-scoped provider use case.
// Usage: Native diagnoseModel resolves the global or conversation credential reference.
// Related: cmd/meca/server_resources.go, internal/provider/diagnostic/check.go.
package server

import (
	"context"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"time"
)

func (s *Service) Diagnose(ctx context.Context, value ModelSettings) (connection.Capabilities, error) {
	if s.options.Diagnose == nil {
		return connection.Capabilities{}, fault.New(diag.FailedPrecondition, diag.UIDiagnosticUnavailable)
	}
	credential, err := parseModelCredential(value.Credential)
	if value.Name == "" || err != nil {
		return connection.Capabilities{}, fault.New(diag.InvalidArgument, diag.ServerInvalidCredential)
	}
	lookup, err := s.beginModelLookup(ctx, value.Session)
	if err != nil {
		return connection.Capabilities{}, err
	}
	defer lookup.finish()
	selected, err := value.selection()
	if err != nil {
		return connection.Capabilities{}, err
	}
	if lookup.previous.Connection.ID == selected.Connection.ID {
		selected.Connection.CredentialRef = lookup.previous.Connection.CredentialRef
	}
	ctx, cancel := context.WithTimeout(lookup.ctx, 60*time.Second)
	defer cancel()
	return s.options.Diagnose(ctx, selected, credential)
}
