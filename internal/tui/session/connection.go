// Purpose: Adopt only durable worker results without performing terminal-loop persistence.
// Pattern: Event-loop-owned state transition.
// Usage: finishWorker applies committed selections and closes rejected candidates.
// Related: internal/tui/session/model_commit.go, internal/tui/session/setup_cancel.go.
package session

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func (a *app) adoptConnection(n notice) (adopted bool, err error) {
	defer func() {
		if !adopted && n.Client != nil && n.Client != a.connection.client {
			n.Client.Close()
		}
	}()
	if n.Commit == nil {
		if n.Client != nil {
			return false, faults.Errorf(diagcatalog.SessionPreparedConnectionHasNoDurableSelection)
		}
		return false, nil
	}
	if n.Commit.Uncertain {
		a.connection.close()
		a.connection.selection = connection.Selection{}
		return false, n.Err
	}
	if n.Err != nil {
		return false, n.Err
	}
	if n.Client == nil || n.Commit.Selection.Model == "" {
		return false, faults.Errorf(diagcatalog.SessionCommittedConnectionHasNoClientOrModel)
	}
	if n.Config != nil {
		a.opts.Config = *n.Config
		a.settings.Permission = n.Config.Permission
	}
	a.connection.adopt(n.Client, n.Commit.Selection)
	a.chat.resetThroughput()
	return true, n.Commit.DefaultError
}
