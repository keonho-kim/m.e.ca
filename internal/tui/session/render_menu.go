// Purpose: Project remaining menus and execution-approval preview scroll positions.
// Pattern: Presentation projection.
// Usage: Session rendering supplies exit and approval choices to view.Menu.
// Related: internal/tui/view/menu.go, internal/tui/view/preview.go.
package session

import "github.com/keonho-kim/meca/internal/tui/view"

func (a *app) renderMenu(w, h int) {
	display := view.MenuDisplay{Activity: a.activity(), Usage: a.displayTokens(), Title: a.language.Text(a.page), Choices: append([]string(nil), a.choices...), Selected: a.choice, Status: a.status, Command: a.menuInput.String()}
	for i, label := range display.Choices {
		display.Choices[i] = a.language.Text(label)
	}
	if a.page == "approval" {
		display.Title = a.language.Text(a.proposal.Name) + " " + a.proposal.Target
		lines := view.PreviewLines(view.ApprovalPreview(*a.proposal, a.language), w, a.proposal.Diff)
		a.previewOffset = min(a.previewOffset, max(0, len(lines)-max(0, h-7)))
		display.Preview = lines[a.previewOffset:]
	}
	view.Menu(a.screen, display, w, h)
}
