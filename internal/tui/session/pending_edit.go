// Purpose: Coordinate reservation editing with chat focus and deferred worker completion.
// Pattern: Application input and completion transitions.
// Usage: Queue keys delegate state changes; closing an edit resumes dispatch or restores interrupted drafts.
// Related: internal/tui/composer/queue/state.go, internal/tui/session/pending.go, internal/tui/session/pending_edit_keys.go.
package session

import "github.com/keonho-kim/meca/internal/tui/composer/queue"

func (a *app) canEditPending() bool {
	return a.page == "chat" && len(a.queued.Messages) > 0 && a.fileSubmission == nil && !a.worker.stopping() && (!a.worker.active() || a.worker.agent())
}

func (a *app) pendingEditChanged() bool { return a.queued.Changed(a.editor.String()) }

func (a *app) navigatePending(delta int) {
	if a.queued.Edit == nil && !a.canEditPending() {
		return
	}
	opening := a.queued.Edit == nil
	switch a.queued.Navigate(delta, &a.editor) {
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
	if !a.queued.Save(a.editor.String()) {
		a.stopNotice = a.language.Text("Enter a request or use Ctrl/Cmd+Backspace to delete it from the queue")
		return
	}
	a.closePendingEdit()
}

func (a *app) deletePendingEdit() { a.queued.DeleteSelected(); a.closePendingEdit() }

func (a *app) restorePendingEditor() queue.Completion {
	if a.queued.Edit == nil {
		return queue.Running
	}
	completion := a.queued.RestoreEditor(&a.editor)
	a.clearSelection()
	a.closeCommandHints()
	a.stopNotice = ""
	return completion
}

func (a *app) closePendingEdit() {
	if a.queued.Edit == nil {
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
