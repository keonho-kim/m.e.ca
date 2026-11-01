// Purpose: Project current measurements and committed token and charge totals without duplication.
// Pattern: Presentation projection.
// Usage: Runtime notices update counters on the terminal event loop.
// Related: internal/tui/session/runtime_events.go, internal/tui/view/tokens.go.
package session

import (
	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (c *chatState) displayTokens() modelapi.TokenUsage {
	return c.tokenTotal.Add(c.tokenCurrent).Add(c.latMemoryTokens)
}
func (c *chatState) receiveTokens(e runrecord.Event) {
	if e.Usage == nil {
		return
	}
	if e.Generation != nil && e.Generation.Role == inference.LATCompaction {
		if e.Kind == "usage" {
			c.tokenTotal = c.tokenTotal.Add(*e.Usage)
			c.latMemoryTokens = modelapi.TokenUsage{}
		} else {
			c.latMemoryTokens = e.Usage.Clone()
		}
		return
	}
	if e.Kind == "usage" {
		c.tokenTotal = c.tokenTotal.Add(*e.Usage)
		c.tokenCurrent = modelapi.TokenUsage{}
	} else {
		c.tokenCurrent = e.Usage.Clone()
	}
	c.tokensPerSecond = e.TokensPerSecond
}
