// Purpose: Project active menus and maintain approval and model-information scroll positions.
// Pattern: Presentation projection.
// Usage: Session render supplies the snapshot to view.Menu.
// Related: internal/tui/modelmenu/identity.go, internal/tui/view/menu.go, internal/tui/view/preview.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/provider/profile"
	"github.com/keonho-kim/v-code/internal/tui/modelmenu"
	"github.com/keonho-kim/v-code/internal/tui/view"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
	"strings"
)

func (a *app) renderMenu(w, h int) {
	display := view.MenuDisplay{Activity: a.activity(), Usage: a.displayTokens(), Title: a.page, Choices: a.choices, Selected: a.choice, Status: a.status, Command: a.menuInput.String()}
	switch a.page {
	case "modelinfo":
		display.Title = "모델 정보 · 최근 연결의 서버 선언"
		lines := viewport.Wrap(strings.Join(modelmenu.IdentityLines(a.selection, a.opts.Config.ContextLength, a.opts.Vault.Redact), "\n"), w)
		a.previewOffset = min(a.previewOffset, max(0, len(lines)-max(1, h-2)))
		display.Details = lines[a.previewOffset:]
	case "model":
		if a.selection.Model != "" {
			display.Model = a.opts.Vault.Redact(a.selection.Model)
			display.IdentifiedModel = modelmenu.IdentifiedName(a.selection, a.opts.Vault.Redact)
			identity := a.selection.Profile
			if identity == "" || identity == profile.Generic {
				identity = "unidentified / generic"
			}
			display.Title = "model · Profile: " + identity
		}
	case "global":
		display.Title = a.changeField + ": " + a.changeValue
	case "key":
		display.Title = "API Key (OS credential store)"
	case "manual":
		display.Title = "Model ID · 검색 또는 직접 입력"
		display.Candidates = a.modelCandidates()
	case "approval":
		display.Title = a.proposal.Name + " " + a.proposal.Target
		lines := view.PreviewLines(a.proposal.Preview, w, a.proposal.Diff)
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
		display.Thinking = &a.thinkingChoice
		display.ThinkingSupported = a.selection.ThinkingSupported
		if info := a.selection.Details; info != nil && info.Reasoning != nil {
			display.ThinkingOnOnly = a.selection.Thinking != nil && *a.selection.Thinking && info.Reasoning.Enabled() && !info.Reasoning.Off
		}
	}
	view.Menu(a.screen, display, w, h)
}
