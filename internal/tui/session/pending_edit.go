// Purpose: Own one queued-message edit and preserve its normal input draft across completion boundaries.
// Pattern: Event-loop state transitions.
// Usage: Queue keys navigate, save or delete; worker completion defers dispatch until editing closes.
// Related: internal/tui/session/pending.go, internal/tui/session/pending_edit_keys.go, internal/tui/session/state.go.
package session

import (
	"strings"

	"github.com/keonho-kim/meca/internal/tui/input"
)

type pendingCompletion uint8

const (
	pendingRunning pendingCompletion = iota
	pendingSucceeded
	pendingStopped
)

type pendingEdit struct {
	index      int
	draft      input.Editor
	completion pendingCompletion
}

func (a *app) canEditPending() bool {
	return a.page == "chat" && len(a.pending) > 0 && a.fileSubmission == nil &&
		!a.worker.stopping() && (!a.worker.active() || a.worker.agent())
}

func (a *app) pendingEditChanged() bool {
	return a.pendingEdit != nil && a.editor.String() != a.pending[a.pendingEdit.index]
}

func (a *app) navigatePending(delta int) {
	if a.pendingEdit == nil {
		if delta >= 0 || !a.canEditPending() {
			return
		}
		a.pendingEdit = &pendingEdit{index: len(a.pending) - 1, draft: a.editor}
		a.clearSelection()
		a.closeCommandHints()
		a.editor.Set(a.pending[a.pendingEdit.index])
		return
	}
	next := a.pendingEdit.index + delta
	if next < 0 {
		return
	}
	if a.pendingEditChanged() {
		a.stopNotice = a.language.Text("Save or cancel changes before switching queued requests")
		return
	}
	if next >= len(a.pending) {
		a.closePendingEdit()
		return
	}
	a.pendingEdit.index = next
	a.editor.Set(a.pending[next])
}

func (a *app) savePendingEdit() {
	text := a.editor.String()
	if strings.TrimSpace(text) == "" {
		a.stopNotice = a.language.Text("Enter a request or use Ctrl+D to cancel its reservation")
		return
	}
	a.pending[a.pendingEdit.index] = text
	a.closePendingEdit()
}

func (a *app) deletePendingEdit() {
	index := a.pendingEdit.index
	copy(a.pending[index:], a.pending[index+1:])
	a.pending[len(a.pending)-1] = ""
	a.pending = a.pending[:len(a.pending)-1]
	if len(a.pending) == 0 {
		a.pending = nil
	}
	a.closePendingEdit()
}

func (a *app) restorePendingEditor() pendingCompletion {
	edit := a.pendingEdit
	if edit == nil {
		return pendingRunning
	}
	a.pendingEdit = nil
	a.editor = edit.draft
	a.clearSelection()
	a.closeCommandHints()
	a.stopNotice = ""
	return edit.completion
}

func (a *app) closePendingEdit() {
	if a.pendingEdit == nil {
		return
	}
	switch a.restorePendingEditor() {
	case pendingSucceeded:
		a.finishPending(true)
	case pendingStopped:
		a.finishPending(false)
	}
	a.refreshHints()
}
