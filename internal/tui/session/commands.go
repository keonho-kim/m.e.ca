// Purpose: Handle submitted local commands and dispatch chat tasks.
// Pattern: Command use case.
// Usage: Session key handlers and command completion call submit.
// Related: internal/commands/parse.go, internal/tui/session/tasks.go, internal/tui/session/menus.go.
package session

import (
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
			a.cancelDirectory()
			a.page = "chat"
			a.editor.Set("")
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
	if a.agentActive && a.busy && command.Kind == commands.Message {
		a.steerInput()
		return
	}
	if !a.canSubmit(command.Kind) {
		a.status = a.language.Text("One task is already active; draft preserved")
		return
	}
	switch command.Kind {
	case commands.Model:
		a.openModel()
	case commands.Directory:
		a.openFiles(command.Argument)
	case commands.Message:
		if a.client == nil || a.selection.Model == "" {
			a.status = a.language.Text("Select a model using !model")
			return
		}
		a.inputs = append(a.inputs, command.Argument)
		if len(a.inputs) > 200 {
			clear(a.inputs[:len(a.inputs)-200])
			a.inputs = a.inputs[len(a.inputs)-200:]
		}
		a.historyIndex = len(a.inputs)
		a.output.Append((transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}).User(a.opts.Vault.Redact(command.Argument)), false)
		a.cancelDirectory()
		a.page = "chat"
		a.startTask(command.Argument)
	}
	a.editor.Set("")
	a.closeCommandHints()
}

func (a *app) canSubmit(kind commands.Kind) bool {
	return !a.busy || kind == commands.Directory || kind == commands.Exit || kind == commands.NewSession
}
