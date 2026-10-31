// Purpose: Handle submitted local commands and dispatch chat tasks.
// Pattern: Command use case.
// Usage: Session key handlers and command completion call submit.
// Related: internal/commands/parse.go, internal/tui/session/tasks.go, internal/tui/session/menus.go.
package session

import (
	"github.com/keonho-kim/meca/internal/commands"
	"github.com/keonho-kim/meca/internal/tui/composer"
)

func (a *app) submit() {
	command, err := a.inputCommand()
	if err != nil {
		a.fail(err)
		return
	}
	if a.page == "files" && a.browser.Input {
		if command.Kind == commands.Message || command.Kind == commands.Empty {
			a.closeFiles()
		}
	}
	a.submitCommand(command)
}

func (a *app) inputCommand() (commands.Command, error) {
	command, err := commands.Parse(a.editor.String())
	if err == nil && a.page == "files" && a.browser.Input && command.Kind == commands.Quit {
		command = commands.Command{Kind: commands.Message, Argument: a.editor.String()}
	}
	return command, err
}

func (a *app) submitCommand(command commands.Command) {
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
		if a.worker.active() {
			a.page = "quit"
			a.choices = []string{"Keep working", "Cancel task and exit"}
			a.choice = 0
			return
		}
		a.exit = true
		return
	}
	if command.Kind == commands.Message && a.fileSelectionBusy {
		a.fileSubmission = &fileSubmission{command: command, revision: a.editor.Revision()}
		a.status = a.language.Text("Saving selected paths before sending")
		return
	}
	a.fileSubmission = nil
	if a.worker.agent() && a.worker.active() && command.Kind == commands.Message {
		a.steerInput(command.Argument)
		return
	}
	if !a.canSubmit(command.Kind) {
		a.status = a.language.Text("One task is already active; draft preserved")
		return
	}
	switch command.Kind {
	case commands.LAT:
		a.lat = !a.lat
		label := "LAT OFF"
		if a.lat {
			label = "LAT ON"
		}
		a.status = a.language.Text(label)
	case commands.Model:
		a.openModel()
	case commands.Language:
		a.openLanguage()
	case commands.Permission:
		a.openPermission()
	case commands.Directory:
		a.openFiles(command.Argument)
	case commands.Message:
		if a.client == nil || a.selection.Model == "" {
			a.status = a.language.Text("Select a model using !model")
			return
		}
		a.cancelDirectory()
		a.page = "chat"
		a.dispatchRequest(composer.Capture(command.Argument, a.selectedPaths()))
	}
	a.editor.Set("")
	a.closeCommandHints()
}

func (a *app) canSubmit(kind commands.Kind) bool {
	return !a.worker.active() || kind == commands.Directory || kind == commands.Exit || kind == commands.NewSession
}
