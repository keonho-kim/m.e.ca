// Purpose: Admit a prepared request without changing the user's foreground browser or draft.
// Pattern: Request dispatch use case.
// Usage: Chat submissions and FIFO completion share connection checks and transcript admission.
// Related: internal/tui/composer/request.go, internal/tui/session/tasks.go, internal/tui/session/pending.go.
package session

import (
	"github.com/keonho-kim/meca/internal/agent/media"
	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/tui/transcript"
)

func (a *app) dispatchRequest(request requestinput.Request) bool {
	if a.connection.client == nil || a.connection.selection.Model == "" {
		a.status = a.language.Text("Select a model using !model")
		return false
	}
	if err := media.CheckInput(request.Images); err != nil {
		a.fail(err)
		return false
	}
	if len(request.Images) > 0 && (a.connection.selection.Details == nil || !a.connection.selection.Details.ImageInput.Supported()) {
		a.status = diagcatalog.Text(diagcatalog.MediaImageInputUnavailable)
		return false
	}
	if a.work.state.active() {
		return false
	}
	a.startRequest(request)
	if !a.work.state.active() {
		return false
	}
	a.compose.rememberInput(request.Text)
	if a.team.enabled {
		a.chat.output.Append("> "+a.opts.Vault.Redact(request.Text)+"\n\n", false)
	} else {
		a.chat.output.Append((transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}).User(a.opts.Vault.Redact(request.Text)), false)
	}
	if len(request.Images) > 0 {
		a.chat.output.Append(a.language.Format("[Images attached: %d]\n\n", len(request.Images)), false)
	}
	return a.work.state.active()
}
