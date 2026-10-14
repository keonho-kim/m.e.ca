// Purpose: Adopt a prepared connection only after its selection has been persisted.
// Pattern: Session settings use case.
// Usage: The event loop calls adoptConnection when a connection worker finishes.
// Related: internal/tui/session/events.go, internal/provider/setup/setup.go, internal/storage/sqlite/selection.go.
package session

import (
	"fmt"
	"github.com/keonho-kim/v-code/internal/provider/setup"
)

func (a *app) adoptConnection(n notice) (err error) {
	adopted := false
	defer func() {
		if !adopted && n.Client != nil && n.Client != a.client {
			n.Client.Close()
		}
	}()
	if n.Err != nil {
		return nil
	}
	if n.Selection == nil {
		if n.Client != nil {
			return fmt.Errorf("prepared client has no selection")
		}
		return nil
	}
	if n.Selection.Model == "" {
		a.pendingConnection = n.Selection.Connection
		a.key = n.credential
		return nil
	}
	if n.Client == nil {
		return fmt.Errorf("prepared selection has no client")
	}
	next := *n.Selection
	if next.Details != nil {
		details := *next.Details
		details.Notice = a.opts.Vault.Redact(details.Notice)
		next.Details = &details
	}
	if next.Identity != nil {
		next.Identity.Redact(a.opts.Vault.Redact)
	}
	next, err = setup.Commit(a.ctx, a.opts.Store, a.opts.Vault, a.opts.Session.ID, next, n.credential)
	if err != nil {
		return err
	}
	previous := a.client
	a.client, a.selection = n.Client, next
	a.key = ""
	adopted = true
	if previous != nil && previous != a.client {
		previous.Close()
	}
	return nil
}
