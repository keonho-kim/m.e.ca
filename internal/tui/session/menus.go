// Purpose: Route local menu choices to model, approval and settings use cases.
// Pattern: Presentation command routing.
// Usage: Invoked by the terminal input handler after a local menu selection.
// Related: internal/tui/session/keys.go, internal/tui/session/model.go, internal/tui/session/settings.go.
package session

import (
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/tui/modelmenu"
)

func (a *app) openModel() {
	a.page = "model"
	a.choices = []string{"Connect provider", "Thinking", modelmenu.EffortLabel(a.selection), "모델 정보"}
	a.choice = 0
	a.thinkingChoice = a.opts.Config.Thinking
	if a.selection.Thinking != nil {
		a.thinkingChoice = *a.selection.Thinking
	}
	a.menuInput.Set("")
}
func (a *app) choose() {
	value := a.choices[a.choice]
	switch a.page {
	case "quit":
		if a.choice == 1 {
			a.requestExit()
		} else {
			a.page = "chat"
		}
	case "approval":
		a.decide(a.choice == 0)
	case "planapproval":
		if a.choice == 0 {
			a.approvePlan()
		} else {
			a.page = "chat"
			a.status = "Enter requested changes"
		}
	case "model":
		if a.choice == 0 {
			if a.busy {
				a.status = "Stop the current task before changing providers"
				return
			}
			a.key = ""
			a.page = "providers"
			a.choices = nil
			for _, provider := range modelconn.Providers() {
				a.choices = append(a.choices, provider.Label)
			}
			a.choice = 0
		} else if a.choice == 1 {
			if !a.selection.ThinkingSupported {
				a.status = "Thinking ON/OFF control is unavailable or unverified for this model"
				return
			}
			a.changeField = "thinking"
			a.changeValue = "OFF"
			if a.thinkingChoice {
				a.changeValue = "ON"
			}
			a.globalMenu()
		} else if a.choice == 2 {
			a.openEffort()
		} else {
			a.page = "modelinfo"
			a.previewOffset = 0
			a.menuInput.Set("")
		}
	case "effort":
		a.chooseEffort()
	case "providers":
		a.connectionKind = a.choice
		a.page = "endpoint"
		c, err := modelconn.New(modelconn.Providers()[a.choice].Kind, "")
		a.fail(err)
		a.menuInput.Set(c.Endpoint)
	case "models":
		if value == "Enter Model ID manually" {
			a.page = "manual"
			a.menuInput.Set("")
			return
		}
		a.connect(a.pendingConnection, a.key, value, true)
		a.key = ""
		a.page = "chat"
	case "permission", "scope":
		a.changeField = a.page
		a.changeValue = value
		a.globalMenu()
	case "global":
		a.applySetting(a.choice == 1)
		a.page = "chat"
	}
}
func (a *app) globalMenu() {
	a.page = "global"
	a.choices = []string{"Current session", "Global override"}
	if a.changeField == "thinking" {
		a.choices[0] = "Current session"
	}
	a.choice = 0
	a.menuInput.Set("")
}
