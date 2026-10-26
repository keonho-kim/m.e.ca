// Purpose: Project current measurements and committed token and charge totals without duplication.
// Pattern: Presentation projection.
// Usage: Runtime notices update counters on the terminal event loop.
// Related: internal/tui/session/runtime_events.go, internal/tui/view/tokens.go.
package session

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (a *app) displayTokens() modelapi.TokenUsage {
	return a.tokenTotal.Add(a.tokenCurrent)
}
func (a *app) receiveTokens(e runrecord.Event) {
	if e.Usage == nil {
		return
	}
	if e.Kind == "usage" {
		a.tokenTotal = a.tokenTotal.Add(*e.Usage)
		a.tokenCurrent = modelapi.TokenUsage{}
	} else {
		a.tokenCurrent = e.Usage.Clone()
	}
	a.tokensPerSecond = e.TokensPerSecond
}
