// Purpose: Route focused keyboard input into local commands, editor actions and menus.
// Pattern: Event Loop input handler.
// Usage: Called only by the terminal loop for tcell key events.
// Related: internal/tui/session/app.go, internal/tui/session/menus.go, internal/tui/input/editor.go, internal/tui/session/hints.go.
package session

import (
	"crypto/rand"
	"time"

	"github.com/gdamore/tcell/v2"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/commands"
	"github.com/keonho-kim/v-code/internal/tui/input"
)

func (a *app) keyEvent(e *tcell.EventKey) {
	if e.Key() == tcell.KeyCtrlC {
		if a.taskCancel != nil {
			a.taskCancel()
			a.status = "Cancelling"
		}
		return
	}
	if a.page == "files" && !a.browser.Input {
		a.fileKey(e)
		return
	}
	if a.page != "chat" && a.page != "files" {
		a.menuKey(e)
		return
	}
	if a.page == "files" && e.Key() == tcell.KeyTAB {
		a.browser.Input = false
		return
	}
	if a.hintKey(e) {
		return
	}
	switch e.Key() {
	case tcell.KeyEnter:
		if a.paste || e.Modifiers()&tcell.ModAlt != 0 {
			a.editor.Insert("\n")
			a.refreshHints()
		} else {
			a.submit()
		}
		return
	case tcell.KeyUp:
		a.recall(-1)
		a.refreshHints()
		return
	case tcell.KeyDown:
		a.recall(1)
		a.refreshHints()
		return
	case tcell.KeyPgUp:
		w, h := a.screen.Size()
		a.scroll(-max(1, h-3), w, h-3)
		return
	case tcell.KeyPgDn:
		w, h := a.screen.Size()
		a.scroll(max(1, h-3), w, h-3)
		return
	}
	editKey(&a.editor, e)
	a.refreshHints()
}
func editKey(editor *input.Editor, e *tcell.EventKey) {
	switch e.Key() {
	case tcell.KeyRune:
		editor.Insert(string(e.Rune()))
	case tcell.KeyLeft:
		editor.Move(-1)
	case tcell.KeyRight:
		editor.Move(1)
	case tcell.KeyHome, tcell.KeyCtrlA:
		editor.Home()
	case tcell.KeyEnd, tcell.KeyCtrlE:
		editor.End()
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		editor.Backspace()
	case tcell.KeyDelete:
		editor.Delete()
	case tcell.KeyCtrlW:
		editor.DeleteWord()
	case tcell.KeyCtrlU:
		editor.Replace(0, editor.Cursor, "")
	case tcell.KeyCtrlK:
		editor.Replace(editor.Cursor, len(editor.Text), "")
	case tcell.KeyTAB:
		editor.Insert("\t")
	}
}
func (a *app) recall(n int) {
	if len(a.inputs) == 0 {
		return
	}
	if a.historyIndex == len(a.inputs) {
		a.draft = a.editor.String()
	}
	a.historyIndex = max(0, min(len(a.inputs), a.historyIndex+n))
	if a.historyIndex == len(a.inputs) {
		a.editor.Set(a.draft)
	} else {
		a.editor.Set(a.inputs[a.historyIndex])
	}
}
func (a *app) submit() {
	command, err := commands.Parse(a.editor.String())
	if err != nil {
		a.fail(err)
		return
	}
	if command.Kind == commands.Empty {
		if a.planMode && a.plan != nil && a.plan.Status == planning.AwaitingApproval {
			a.page = "planapproval"
			a.choices = []string{"Approve", "Request changes"}
			a.choice = 0
		}
		return
	}
	if command.Kind == commands.Quit {
		if a.page == "files" {
			a.page = "chat"
			a.editor.Set("")
			return
		}
		if a.planMode {
			a.status = "Plan approval is required"
			return
		}
		if a.busy {
			a.page = "quit"
			a.choices = []string{"Keep working", "Cancel task and exit"}
			a.choice = 0
			return
		}
		a.exit = true
		return
	}
	if a.busy && command.Kind != commands.Directory {
		a.status = "One task is already active; draft preserved"
		return
	}
	switch command.Kind {
	case commands.Model:
		a.openModel()
	case commands.Permission:
		a.page = "permission"
		a.choices = []string{"approve", "allow-all"}
		a.choice = 0
	case commands.Scope:
		a.page = "scope"
		a.choices = []string{"directory", "system"}
		a.choice = 0
	case commands.Directory:
		a.openFiles(command.Argument)
	case commands.Plan:
		if !a.planMode {
			p, err := planning.NewPlan("plan_"+rand.Text(), a.opts.Session.ID, "Awaiting objective", []string{"Define the task objective"}, time.Now())
			if err != nil {
				a.fail(err)
				return
			}
			p.Status = planning.Planning
			if err = a.opts.Store.SavePlan(a.ctx, p); err != nil {
				a.fail(err)
				return
			}
			a.plan = &p
			a.planMode = true
		}
	case commands.Message:
		if a.client == nil || a.selection.Model == "" {
			a.status = "Select a model using !model"
			return
		}
		a.inputs = append(a.inputs, command.Argument)
		if len(a.inputs) > 200 {
			a.inputs = a.inputs[len(a.inputs)-200:]
		}
		a.historyIndex = len(a.inputs)
		a.output.Append("> "+a.opts.Vault.Redact(command.Argument), false)
		a.page = "chat"
		a.startTask(command.Argument, nil)
	}
	a.editor.Set("")
	a.closeReferences()
	a.closeCommandHints()
}
func (a *app) menuKey(e *tcell.EventKey) {
	if a.page == "approval" {
		if e.Key() == tcell.KeyPgDn {
			a.previewOffset += 8
			return
		}
		if e.Key() == tcell.KeyPgUp {
			a.previewOffset = max(0, a.previewOffset-8)
			return
		}
	}
	if a.paste {
		if e.Key() == tcell.KeyRune {
			a.menuInput.Insert(string(e.Rune()))
		}
		return
	}
	if e.Key() == tcell.KeyEscape {
		if a.page == "approval" {
			a.decide(false)
		} else {
			a.page = "chat"
		}
		a.menuInput.Set("")
		return
	}
	if a.page == "endpoint" || a.page == "key" || a.page == "manual" {
		if e.Key() == tcell.KeyEnter {
			if a.menuInput.String() == ":q" {
				a.page = "chat"
				a.menuInput.Set("")
				return
			}
			a.confirmField()
			return
		}
		editKey(&a.menuInput, e)
		return
	}
	switch e.Key() {
	case tcell.KeyUp:
		a.choice = max(0, a.choice-1)
	case tcell.KeyDown:
		a.choice = min(max(0, len(a.choices)-1), a.choice+1)
	case tcell.KeyEnter:
		if a.menuInput.String() == ":q" {
			if a.page == "approval" {
				a.decide(false)
			} else {
				a.page = "chat"
			}
			a.menuInput.Set("")
			return
		}
		if len(a.choices) > 0 {
			a.choose()
		}
	default:
		editKey(&a.menuInput, e)
	}
}
func (a *app) decide(approved bool) {
	select {
	case a.decision <- approved:
	default:
	}
	a.proposal = nil
	a.page = "chat"
}
