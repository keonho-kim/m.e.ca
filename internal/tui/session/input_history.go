// Purpose: Bound recalled input identically for direct requests and delivered direction changes.
// Pattern: Event-loop state mutation.
// Usage: Submission and runtime message acknowledgement append through rememberInput.
// Related: internal/tui/session/commands.go, internal/tui/session/runtime_events.go, internal/tui/session/keys.go.
package session

import "github.com/keonho-kim/meca/internal/storage/history"

func (r *requestState) rememberInput(text string) {
	r.inputs = append(r.inputs, text)
	bytes := 0
	for _, input := range r.inputs {
		bytes += len(input)
	}
	for len(r.inputs) > 0 && (len(r.inputs) > history.MaxRecentInputs || bytes > history.RecentInputBytes) {
		bytes -= len(r.inputs[0])
		r.inputs[0] = ""
		r.inputs = r.inputs[1:]
	}
	r.historyIndex = len(r.inputs)
}

func (r *requestState) recall(n int) {
	if len(r.inputs) == 0 {
		return
	}
	if r.historyIndex == len(r.inputs) {
		r.draft = r.editor.String()
	}
	r.historyIndex = max(0, min(len(r.inputs), r.historyIndex+n))
	if r.historyIndex == len(r.inputs) {
		r.editor.Set(r.draft)
	} else {
		r.editor.Set(r.inputs[r.historyIndex])
	}
}
