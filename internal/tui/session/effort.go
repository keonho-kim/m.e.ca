// Purpose: Select and persist effort choices advertised by server metadata or the serving API.
// Pattern: Event-loop settings use case.
// Usage: The model menu saves an effort choice; explicit connection diagnostics probe it separately.
// Related: internal/provider/selection/effort.go, internal/tui/session/diagnostic.go, internal/storage/sqlite/selection.go.
package session

import (
	"fmt"
	selectionpolicy "github.com/keonho-kim/v-code/internal/provider/selection"
	"strings"
)

func (a *app) openEffort() {
	values := selectionpolicy.EffortValues(a.selection)
	if len(values) == 0 {
		a.status = a.language.Text("Thinking effort is unavailable for this connection")
		return
	}
	a.page = "effort"
	a.choices = []string{"DEFAULT"}
	a.choice = 0
	for _, value := range values {
		a.choices = append(a.choices, strings.ToUpper(value))
		if value == a.selection.Effort() {
			a.choice = len(a.choices) - 1
		}
	}
	a.menuInput.Set("")
}

func (a *app) chooseEffort() {
	value := ""
	if a.choice > 0 {
		values := selectionpolicy.EffortValues(a.selection)
		if a.choice > len(values) {
			a.status = a.language.Text("Reasoning choices changed; reopen the menu")
			return
		}
		value = values[a.choice-1]
	}
	if a.busy {
		a.status = a.language.Text("Stop the current task before changing Thinking effort")
		return
	}
	if a.client == nil {
		a.status = a.language.Text("Select a model using !model")
		return
	}
	if err := a.applyEffort(value); err != nil {
		a.fail(err)
		return
	}
	a.openModel()
}

func (a *app) applyEffort(value string) error {
	next := a.selection
	if next.Thinking == nil && value != "" {
		info := next.Details
		if info == nil || info.Reasoning == nil || !info.Reasoning.Enabled() || info.Reasoning.Off {
			return fmt.Errorf("%s", a.language.Text("Select Thinking ON or OFF before choosing effort"))
		}
		on := true
		next.Thinking = &on
	}
	next.ReasoningEffort = &value
	if err := a.opts.Store.SaveSelection(a.ctx, a.opts.Session.ID, next); err != nil {
		return err
	}
	a.selection = next
	return nil
}
