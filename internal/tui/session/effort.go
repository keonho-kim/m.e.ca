// Purpose: Present model-specific effort values and apply only successfully probed and persisted choices.
// Pattern: Settings use case with cancellable worker lifecycle.
// Usage: The model menu opens effort choices; notices apply confirmed preferences in the event loop.
// Related: internal/provider/profile/effort.go, internal/provider/diagnostic/effort.go, internal/tui/session/events.go.
package session

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/provider/diagnostic"
	selectionpolicy "github.com/keonho-kim/v-code/internal/provider/selection"
	"strings"
)

func (a *app) openEffort() {
	values := selectionpolicy.EffortValues(a.selection)
	if len(values) == 0 {
		a.status = "Thinking effort is unavailable or undocumented for this model and provider"
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
			a.status = "Reasoning choices changed; reopen the menu"
			return
		}
		value = values[a.choice-1]
	}
	if a.busy {
		a.status = "Stop the current task before changing Thinking effort"
		return
	}
	if a.client == nil {
		a.status = "Select a model using !model"
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.taskCancel = cancel
	a.busy = true
	a.startElapsed()
	a.status = "Checking Thinking effort"
	a.page = "chat"
	client, model := a.client, a.selection.Model
	inferenceSettings := a.opts.Config.Inference
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		err := diagnostic.Effort(ctx, client, model, value, &inferenceSettings)
		a.post(a.ctx, notice{Done: true, Err: err, Effort: &value})
	}()
}

func (a *app) applyEffort(value string) error {
	if a.selection.Thinking == nil && !a.selection.ThinkingSupported && value != "" {
		return fmt.Errorf("Thinking ON/OFF control is unverified")
	}
	next := a.selection
	next.ReasoningEffort = &value
	if err := a.opts.Store.SaveSelection(a.ctx, a.opts.Session.ID, next); err != nil {
		return err
	}
	a.selection = next
	return nil
}
