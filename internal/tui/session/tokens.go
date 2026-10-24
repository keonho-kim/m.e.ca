// Purpose: Project measured usage from the current generation and committed call totals.
// Pattern: Presentation projection.
// Usage: Runtime notices update counters on the terminal event loop.
// Related: internal/tui/session/runtime_events.go, internal/tui/view/tokens.go.
package session

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/run"
)

func (a *app) displayTokens() modelapi.TokenUsage {
	return modelapi.TokenUsage{Input: a.tokenTotal.Input + a.tokenCurrent.Input, Output: a.tokenTotal.Output + a.tokenCurrent.Output}
}
func (a *app) receiveTokens(e run.Event) {
	if e.Usage == nil {
		return
	}
	if e.Kind == "usage" {
		a.tokenTotal.Input += e.Usage.Input
		a.tokenTotal.Output += e.Usage.Output
		a.tokenCurrent = modelapi.TokenUsage{}
	} else {
		a.tokenCurrent = *e.Usage
	}
	a.tokensPerSecond = e.TokensPerSecond
}
