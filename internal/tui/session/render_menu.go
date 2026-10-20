// Purpose: Project active menus and maintain approval and model-information scroll positions.
// Pattern: Presentation projection.
// Usage: Session render supplies the snapshot to view.Menu.
// Related: internal/tui/modelmenu/identity.go, internal/tui/view/menu.go, internal/tui/view/preview.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/tui/modelmenu"
	"github.com/keonho-kim/v-code/internal/tui/view"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
	"strings"
)

func (a *app) renderMenu(w, h int) {
	display := view.MenuDisplay{Language: a.language, Activity: a.activity(), Usage: a.displayTokens(), Title: a.language.Text(a.page), Choices: append([]string(nil), a.choices...), Selected: a.choice, Status: a.status, Command: a.menuInput.String()}
	for i, label := range display.Choices {
		display.Choices[i] = a.language.Text(label)
	}
	switch a.page {
	case "modelinfo":
		display.Title = a.language.Text("Model information · latest server declarations")
		lines := viewport.Wrap(strings.Join(modelmenu.IdentityLines(a.selection, a.opts.Config.ContextLength, a.opts.Vault.Redact, a.language), "\n"), w)
		a.previewOffset = min(a.previewOffset, max(0, len(lines)-max(1, h-2)))
		display.Details = lines[a.previewOffset:]
	case "model":
		if a.selection.Model != "" {
			display.Model = a.opts.Vault.Redact(a.selection.Model)
			display.IdentifiedModel = modelmenu.IdentifiedName(a.selection, a.opts.Vault.Redact)
			if display.Status == "" && a.selection.ThinkingSupported && (a.selection.Details == nil || a.selection.Details.Reasoning == nil) {
				display.Status = a.language.Text("API options; run connection diagnostics to verify this model")
			}

		}
	case "global":
		display.Title = a.language.Text("thinking") + ": " + a.language.Text(a.changeValue)
	case "key":
		display.Title = a.language.Text("API Key (OS credential store)")
	case "manual":
		display.Title = a.language.Text("Model ID · search or type")
		display.Candidates = a.modelCandidates()
	case "approval":
		display.Title = a.language.Text(a.proposal.Name) + " " + a.proposal.Target
		lines := view.PreviewLines(view.ApprovalPreview(*a.proposal, a.language), w, a.proposal.Diff)
		a.previewOffset = min(a.previewOffset, max(0, len(lines)-max(0, h-7)))
		display.Preview = lines[a.previewOffset:]
	}
	if a.page == "endpoint" || a.page == "key" || a.page == "manual" {
		window := a.menuInput.FieldWindow(w, a.page == "key")
		display.Field = &window
		if a.page != "key" {
			a.inputRegion = inputRegion{editor: &a.menuInput, window: window, y: 2, width: w, revision: a.menuInput.Revision()}
		}
	}
	if a.page == "model" {
		display.ThinkingChoice = true
		display.Thinking = a.thinkingChoice
		display.ThinkingSupported = a.selection.ThinkingSupported
		if info := a.selection.Details; info != nil && info.Reasoning != nil {
			display.ThinkingOnOnly = info.Reasoning.Enabled() && !info.Reasoning.Off
		}
	}
	view.Menu(a.screen, display, w, h)
}
