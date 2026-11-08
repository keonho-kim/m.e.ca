// Purpose: Render bounded skill preview windows and rebuild layout after geometry changes.
// Pattern: Terminal projection with one cancellable layout worker.
// Usage: The session renderer calls renderSkills on its event loop.
// Related: internal/tui/session/skills.go, internal/tui/view/preview.go.
package session

import "github.com/keonho-kim/meca/internal/tui/view"

func (a *app) renderSkills(w, h int) {
	display := view.MenuDisplay{Activity: a.activity(), Usage: a.chat.displayTokens(), Title: a.language.Text("Skills"), Choices: a.choices, Selected: a.choice, Status: a.status}
	if doc := a.skills.document; doc != nil {
		if a.skills.width != w && !a.skills.busy {
			ctx, job := a.beginSkillJob()
			document := *doc
			a.workers.Add(1)
			go func() {
				defer a.workers.Done()
				lines, err := view.BuildPreview(ctx, document.Body, max(12, w), false)
				_ = a.post(a.ctx, notice{Skills: &skillNotice{document: &document, preview: lines, width: w, err: err, job: job, layoutOnly: true}})
			}()
		}
		display.Title = doc.Name + " [" + doc.Source + "]"
		a.previewOffset = min(a.previewOffset, max(0, a.skills.preview.Count()-max(1, h-7)))
		display.Preview = a.skills.preview.Window(a.previewOffset, max(1, h-7))
	}
	a.headerTitle = display.Title
	view.Menu(a.screen, display, w, h)
}
