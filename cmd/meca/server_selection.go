// Purpose: Resolve persisted server models before configured defaults for reads and execution.
// Pattern: Pure selection policy shared by the concrete session adapters.
// Usage: Read-only and writable conversation factories call this without connecting a provider.
// Related: cmd/meca/server_resources.go, cmd/meca/server_session.go.
package main

import (
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func serverSelection(selected connection.Selection, fallback *config.SavedModel) (connection.Selection, error) {
	if selected.Model != "" || fallback == nil {
		return selected, nil
	}
	return fallback.Selection()
}
