// Purpose: Bound recalled input identically for direct requests and delivered direction changes.
// Pattern: Event-loop state mutation.
// Usage: Submission and runtime message acknowledgement append through rememberInput.
// Related: internal/tui/session/commands.go, internal/tui/session/runtime_events.go, internal/tui/session/keys.go.
package session

func (a *app) rememberInput(text string) {
	a.inputs = append(a.inputs, text)
	if len(a.inputs) > 200 {
		clear(a.inputs[:len(a.inputs)-200])
		a.inputs = a.inputs[len(a.inputs)-200:]
	}
	a.historyIndex = len(a.inputs)
}
