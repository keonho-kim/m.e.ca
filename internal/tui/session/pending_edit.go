// Purpose: Coordinate reservation editing with chat focus and deferred worker completion.
// Pattern: Application input and completion transitions.
// Usage: Queue keys delegate state changes; closing an edit resumes dispatch or restores interrupted drafts.
// Related: internal/tui/composer/queue/state.go, internal/tui/session/pending.go, internal/tui/session/pending_edit_keys.go.
package session

import (
	"github.com/keonho-kim/meca/internal/tui/composer/queue"
)

func (a *app) canEditPending() bool {
	return a.page == "chat" && len(a.compose.queued.Messages) > 0 && a.compose.fileSubmission == nil && !a.work.state.stopping() && (!a.work.state.active() || a.work.state.agent())
}

func (a *app) pendingEditChanged() bool { return a.compose.queued.Changed(a.compose.editor.String()) }

func (a *app) navigatePending(delta int) {
	if a.compose.queued.Edit == nil && !a.canEditPending() {
		return
	}
	opening := a.compose.queued.Edit == nil
	switch a.compose.queued.Navigate(delta, &a.compose.editor) {
	case queue.Focused:
		if opening {
			a.clearSelection()
			a.closeCommandHints()
		}
	case queue.Unsaved:
		a.stopNotice = a.language.Text("Save or cancel changes before switching queued requests")
	case queue.Close:
		a.closePendingEdit()
	}
}

func (a *app) savePendingEdit() {
	if !a.compose.queued.Save(a.compose.editor.String()) {
		a.stopNotice = a.language.Text("Enter a request or use Ctrl/Cmd+Backspace to delete it from the queue")
		return
	}
	a.closePendingEdit()
}

func (a *app) deletePendingEdit() { a.compose.queued.DeleteSelected(); a.closePendingEdit() }

func (a *app) restorePendingEditor() queue.Completion {
	if a.compose.queued.Edit == nil {
		return queue.Running
	}
	completion := a.compose.queued.RestoreEditor(&a.compose.editor)
	a.clearSelection()
	a.closeCommandHints()
	a.stopNotice = ""
	return completion
}

func (a *app) closePendingEdit() {
	if a.compose.queued.Edit == nil {
		return
	}
	switch a.restorePendingEditor() {
	case queue.Succeeded:
		a.finishPending(true)
	case queue.Stopped:
		a.finishPending(false)
	}
	a.refreshHints()
}
