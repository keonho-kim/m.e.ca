// Purpose: Describe the last successfully selected connection for sessions without their own model.
// Pattern: Configuration value boundary.
// Usage: Setup remembers non-secret connection fields; startup rebuilds an unverified selection.
// Related: internal/config/settings.go, internal/tui/session/session.go, internal/provider/setup/commit.go.
package config

import (
	"fmt"
	"strings"

	"github.com/keonho-kim/v-code/internal/provider/connection"
)

type SavedModel struct {
	Provider      connection.ProviderKind `toml:"provider"`
	Endpoint      string                  `toml:"endpoint"`
	Name          string                  `toml:"name"`
	CredentialRef string                  `toml:"credential_ref,omitempty"`
}

func RememberModel(selected connection.Selection) *SavedModel {
	return &SavedModel{Provider: selected.Connection.Kind, Endpoint: selected.Connection.Endpoint, Name: selected.Model, CredentialRef: selected.Connection.CredentialRef}
}

func (m SavedModel) Selection() (connection.Selection, error) {
	conn, err := connection.New(m.Provider, m.Endpoint)
	if err != nil {
		return connection.Selection{}, err
	}
	if strings.TrimSpace(m.Name) == "" || len(m.Name) > 512 || strings.ContainsAny(m.Name, "\r\n\x00") || len(m.CredentialRef) > 128 || strings.ContainsAny(m.CredentialRef, "\r\n\x00") {
		return connection.Selection{}, fmt.Errorf("invalid saved model name or credential reference")
	}
	conn.CredentialRef = m.CredentialRef
	return connection.Selection{Connection: conn, Model: m.Name}, nil
}
