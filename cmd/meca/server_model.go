//go:build !minimum

// Purpose: Prepare an owned server model client without persisting selections or credentials.
// Pattern: Composition root model I/O boundary.
// Usage: Server idle-only model commands call prepareServerModel and adopt its client.
// Related: internal/provider/setup/setup.go, internal/commands/server/settings.go, cmd/meca/server.go.
package main

import (
	"context"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
)

func prepareServerModel(ctx context.Context, next connection.Selection, credential string, defaults setup.Defaults, vault *credentials.Vault) (server.PreparedModel, error) {
	if err := vault.Remember(credential); err != nil {
		return server.PreparedModel{}, err
	}
	prepared, err := setup.Prepare(ctx, next.Connection, credential, next.Model, defaults, next, vault)
	if err != nil {
		return server.PreparedModel{}, err
	}
	next = *prepared.Selection
	redactServerSelection(&next, vault)

	return server.PreparedModel{Selection: next, Generator: prepared.Client, Close: prepared.Client.Close}, nil
}

func redactServerSelection(selected *connection.Selection, vault *credentials.Vault) {
	if selected.Identity != nil {
		selected.Identity.Redact(vault.Redact)
	}
	if selected.Details != nil {
		selected.Details.Notice = vault.Redact(selected.Details.Notice)
	}
}
