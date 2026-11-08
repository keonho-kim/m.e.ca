// Purpose: Browse and preview skills with one owned background read and event-loop state.
// Pattern: Terminal read adapter.
// Usage: !skills opens the menu; previews never submit model work.
// Related: internal/tui/session/skills_hints.go, internal/tui/session/events.go, internal/skills/reader.go.
package session

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/skills/instruction"
	"github.com/keonho-kim/meca/internal/tui/view"
)

type skillBrowserState struct {
	page        skills.Page
	document    *instruction.Document
	preview     *view.Preview
	busy        bool
	job         uint64
	cancel      context.CancelFunc
	hintBusy    bool
	hintJob     uint64
	hintCancel  context.CancelFunc
	width       int
	openPending bool
}
type skillNotice struct {
	job         uint64
	layoutOnly  bool
	page        *skills.Page
	document    *instruction.Document
	preview     *view.Preview
	suggestions []instruction.Metadata
	query       string
	err         error
	width       int
}

func (a *app) skillReader() *skills.Reader {
	return a.opts.Skills
}

func (a *app) beginSkillJob() (context.Context, uint64) {
	a.skills.busy = true
	a.skills.job++
	ctx, cancel := context.WithCancel(a.ctx)
	a.skills.cancel = cancel
	return ctx, a.skills.job
}

func (a *app) closeSkills() {
	if a.skills.cancel != nil {
		a.skills.cancel()
	}
	if a.skills.hintCancel != nil {
		a.skills.hintCancel()
	}
	a.skills.document, a.skills.preview = nil, nil
	a.skills.page = skills.Page{}
	a.skills.openPending = false
	a.skills.width = 0
}

func (a *app) openSkills() {
	a.page, a.previewOffset, a.choice = "skills", 0, 0
	a.skills.document, a.skills.preview = nil, nil
	if a.skills.hintCancel != nil {
		a.skills.hintCancel()
	}
	if a.skills.busy {
		a.skills.openPending = true
		if a.skills.cancel != nil {
			a.skills.cancel()
		}
		return
	}
	a.loadSkillPage("")
}

func (a *app) loadSkillPage(cursor string) {
	reader := a.skillReader()
	if reader == nil {
		a.status = a.language.Text("Skill reader is unavailable")
		return
	}
	if a.skills.busy {
		return
	}
	ctx, job := a.beginSkillJob()
	a.status, a.choices = a.language.Text("Reading skills"), nil
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		page, err := reader.List(ctx, cursor, 32)
		// Another traversal can outlive the bounded snapshot cache. Restart
		// once from current metadata instead of leaving an unusable menu.
		if cursor != "" && errors.Is(err, skills.ErrInvalid) && ctx.Err() == nil {
			page, err = reader.List(ctx, "", 32)
		}
		_ = a.post(a.ctx, notice{Skills: &skillNotice{page: &page, err: err, job: job}})
	}()
}

func (a *app) chooseSkill() {
	if a.skills.busy {
		return
	}
	if a.skills.document != nil {
		a.skills.document, a.skills.preview = nil, nil
		a.openSkills()
		return
	}
	if a.choice == len(a.skills.page.Items) && a.skills.page.NextCursor != "" {
		a.loadSkillPage(a.skills.page.NextCursor)
		return
	}
	if a.choice < 0 || a.choice >= len(a.skills.page.Items) {
		return
	}
	reader := a.skillReader()
	if reader == nil {
		return
	}
	id := a.skills.page.Items[a.choice].ID
	width, _ := a.screen.Size()
	ctx, job := a.beginSkillJob()
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		doc, err := reader.Load(ctx, instruction.Selector{ID: id}, true)
		var lines *view.Preview
		if err == nil {
			lines, err = view.BuildPreview(ctx, doc.Body, max(12, width), false)
		}
		_ = a.post(a.ctx, notice{Skills: &skillNotice{document: &doc, preview: lines, err: err, width: width, job: job}})
	}()
}

func (a *app) receiveSkills(n *skillNotice) {
	if n.query != "" {
		if n.job != a.skills.hintJob {
			return
		}
		a.skills.hintBusy = false
		if a.skills.hintCancel != nil {
			a.skills.hintCancel()
			a.skills.hintCancel = nil
		}
		a.receiveSkillHints(n)
		return
	}
	if n.job != a.skills.job {
		return
	}
	a.skills.busy = false
	if a.skills.cancel != nil {
		a.skills.cancel()
		a.skills.cancel = nil
	}

	if a.skills.openPending {
		a.skills.openPending = false
		if a.page == "skills" {
			a.openSkills()
			return
		}
	}
	if a.page != "skills" {
		return
	}
	if n.err != nil {
		a.fail(n.err)
		return
	}
	if !n.layoutOnly {
		a.status, a.previewOffset, a.choice = "", 0, 0
	}
	if n.document != nil {
		a.skills.document, a.skills.preview = n.document, n.preview
		a.skills.width = n.width
		a.choices = []string{a.language.Text("Back to skills")}
		a.status = strings.Join(n.document.Warnings, " ")
		return
	}
	if n.page == nil {
		return
	}
	a.skills.page = *n.page
	a.choices = nil
	for _, m := range n.page.Items {
		a.choices = append(a.choices, fmt.Sprintf("%s [%s] — %s", m.Name, m.Source, m.Description))
	}
	if n.page.NextCursor != "" {
		a.choices = append(a.choices, a.language.Text("More skills"))
	}
	for _, d := range n.page.Diagnostics {
		a.choices = append(a.choices, "! "+d.ID+": "+d.Message)
	}
	if len(n.page.Diagnostics) > 0 {
		a.status = fmt.Sprintf("%s: %d", a.language.Text("Invalid or inaccessible skills"), len(n.page.Diagnostics))
	}
	if len(a.choices) == 0 {
		a.status = a.language.Text("No skills found")
	}
}
