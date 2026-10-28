// Purpose: Admit a prepared request without changing the user's foreground browser or draft.
// Pattern: Request dispatch use case.
// Usage: Chat submissions and FIFO completion share connection checks and transcript admission.
// Related: internal/tui/composer/request.go, internal/tui/session/tasks.go, internal/tui/session/pending.go.
package session

import (
	"github.com/keonho-kim/meca/internal/tui/composer"
	"github.com/keonho-kim/meca/internal/tui/transcript"
)

func (a *app) dispatchRequest(request composer.Request) {
	if a.client == nil || a.selection.Model == "" {
		a.status = a.language.Text("Select a model using !model")
		return
	}
	a.rememberInput(request.Text)
	a.output.Append((transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}).User(a.opts.Vault.Redact(request.Text)), false)
	a.startRequest(request)
}
