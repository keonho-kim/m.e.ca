// Purpose: Handle submitted local commands and dispatch chat tasks.
// Pattern: Command use case.
// Usage: Session key handlers and command completion call submit.
// Related: internal/commands/parse.go, internal/tui/session/tasks.go, internal/tui/session/menus.go.
package session

import (
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/commands"
	"github.com/keonho-kim/v-code/internal/tui/transcript"
)

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
	if command.Kind == commands.Exit {
		a.requestExit()
		return
	}
	if command.Kind == commands.NewSession {
		a.newSession = true
		a.requestExit()
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
	if a.busy && command.Kind != commands.Directory && !(command.Kind == commands.Plan && a.planMode && a.taskCancel != nil) {
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
		if err := a.togglePlan(); err != nil {
			a.fail(err)
			return
		}
	case commands.Message:
		if a.client == nil || a.selection.Model == "" {
			a.status = "Select a model using !model"
			return
		}
		a.inputs = append(a.inputs, command.Argument)
		if len(a.inputs) > 200 {
			clear(a.inputs[:len(a.inputs)-200])
			a.inputs = a.inputs[len(a.inputs)-200:]
		}
		a.historyIndex = len(a.inputs)
		a.output.Append(transcript.User(a.opts.Vault.Redact(command.Argument)), false)
		a.page = "chat"
		a.startTask(command.Argument, nil)
	}
	a.editor.Set("")
	a.closeReferences()
	a.closeCommandHints()
}
