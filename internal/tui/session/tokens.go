// Purpose: Project current measurements and committed token and charge totals without duplication.
// Pattern: Presentation projection.
// Usage: Runtime notices update counters on the terminal event loop.
// Related: internal/tui/session/runtime_events.go, internal/tui/view/tokens.go.
package session

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (c *chatState) displayTokens() modelapi.TokenUsage {
	return c.tokenTotal.Add(c.tokenCurrent)
}
func (c *chatState) receiveTokens(e runrecord.Event) {
	if e.Usage == nil {
		return
	}
	if e.Kind == "usage" {
		c.tokenTotal = c.tokenTotal.Add(*e.Usage)
		c.tokenCurrent = modelapi.TokenUsage{}
		c.measured = c.measured.Add(runrecord.MeasuredThroughput(e))
		c.currentMeasurement = runrecord.Throughput{}
	} else {
		c.tokenCurrent = e.Usage.Clone()
		c.currentMeasurement = runrecord.MeasuredThroughput(e)
	}
	if rate := c.measured.Add(c.currentMeasurement).Rate(); rate != nil {
		c.tokensPerSecond = rate
	}
}

func (c *chatState) resetThroughput() {
	c.measured, c.currentMeasurement = runrecord.Throughput{}, runrecord.Throughput{}
	c.tokensPerSecond = nil
}
