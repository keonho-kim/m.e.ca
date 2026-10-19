// Purpose: Cancel connection diagnostics and reject results from superseded setup requests.
// Pattern: Worker lifecycle boundary.
// Usage: Esc cancels setup; receive checks the request identity before adopting a client.
// Related: internal/tui/session/model.go, internal/tui/session/effort.go, internal/tui/session/events.go.
package session

import "context"

func (a *app) interruptSetup() {
	if a.taskCancel == nil {
		return
	}
	a.stopRequested = true
	a.taskCancel()
	a.stopElapsed()
	a.key = ""
	a.closeModelHints()
	a.page = "chat"
	a.status = ""
	a.stopNotice = a.language.Text("연결 작업을 취소하고 정리하고 있습니다")
}

func (a *app) setupResult(n *notice) bool {
	if n.SetupRevision == 0 {
		return true
	}
	if n.SetupRevision != a.setupRevision || a.stopRequested {
		if n.Client != nil && n.Client != a.client {
			n.Client.Close()
		}
		n.Client, n.Selection, n.Models, n.credential = nil, nil, nil, ""
		if n.SetupRevision != a.setupRevision {
			return false
		}
		n.Err = context.Canceled
	}
	return true
}
