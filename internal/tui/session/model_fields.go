// Purpose: Apply endpoint, credential and model-ID field transitions.
// Pattern: Presentation input handler.
// Usage: Called by the focused menu input router.
// Related: internal/tui/session/model.go, internal/tui/session/menu_keys.go.
package session

import (
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"strings"
)

func (a *app) confirmField() {
	switch a.page {
	case "endpoint":
		endpoint := strings.TrimSpace(a.menuInput.String())
		if endpoint == "" {
			a.status = "Endpoint cannot be empty"
			return
		}
		connection, err := modelconn.New(modelconn.Providers()[a.connectionKind].Kind, endpoint)
		if err != nil {
			a.fail(err)
			return
		}
		a.endpoint = connection.Endpoint
		a.status = ""
		a.page = "key"
		a.menuInput.Set("")
	case "key":
		a.key = a.menuInput.String()
		a.opts.Vault.Remember(a.key)
		connection, err := modelconn.New(modelconn.Providers()[a.connectionKind].Kind, a.endpoint)
		if err != nil {
			a.fail(err)
			return
		}
		a.menuInput.Set("")

		a.connect(connection, a.key, "", false)
		a.key = ""
		a.page = "chat"
	case "manual":
		model := strings.TrimSpace(a.menuInput.String())
		if model == "" {
			return
		}
		a.connect(a.pendingConnection, a.key, model, true)
		a.key = ""
		a.menuInput.Set("")
		a.page = "chat"
	}
}
