// Purpose: Explicitly diagnose a connection without saving its model or credentials.
// Pattern: Request-scoped provider use case.
// Usage: Native diagnoseModel borrows a conversation for its saved credential reference.
// Related: cmd/meca/server_resources.go, internal/provider/diagnostic/check.go.
package server

import (
	"context"
	"github.com/keonho-kim/meca/internal/config"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"strings"
	"time"
)

func (s *Service) Diagnose(ctx context.Context, value ModelSettings) (connection.Capabilities, error) {
	if s.options.Diagnose == nil {
		return connection.Capabilities{}, fault.New(diag.FailedPrecondition, diag.UIDiagnosticUnavailable)
	}
	if value.Name == "" || len(value.Credential) > 2048 || strings.ContainsAny(value.Credential, "\r\n\x00") {
		return connection.Capabilities{}, fault.New(diag.InvalidArgument, diag.ServerInvalidCredential)
	}
	c, ctx, done, err := s.beginSessionRead(ctx, value.Session)
	if err != nil {
		return connection.Capabilities{}, err
	}
	defer done()
	selected, err := (config.SavedModel{Provider: connection.ProviderKind(value.Provider), Endpoint: value.Endpoint, Name: value.Name}).Selection()
	if err != nil {
		return connection.Capabilities{}, err
	}
	s.mu.Lock()
	previous := c.Runtime.Selection
	s.mu.Unlock()
	if previous.Connection.ID == selected.Connection.ID {
		selected.Connection.CredentialRef = previous.Connection.CredentialRef
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	return s.options.Diagnose(ctx, selected, value.Credential)
}
