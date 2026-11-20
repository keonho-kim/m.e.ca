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
	return c.tokenTotal.Add(c.tokenCurrent)
}
func (c *chatState) receiveTokens(e runrecord.Event) {
	if e.Usage == nil {
		return
	}
	if e.Kind == "usage" {
		c.tokenTotal = c.tokenTotal.Add(*e.Usage)
		c.tokenCurrent = modelapi.TokenUsage{}
	} else {
		c.tokenCurrent = e.Usage.Clone()
	}
}
func (c *chatState) updateAttempt(e *runrecord.Event) {
	if e.Kind == "generation_started" || e.Kind == "team_notice" && e.Text == "left" {
		delete(c.attempts, e.Agent)
	}
	if e.Activity == nil {
		return
	}
	if e.Activity.Role != inference.Agent {
		delete(c.attempts, e.Agent)
		return
	}
	if c.attempts == nil {
		c.attempts = make(map[string]runrecord.ModelActivity)
	}
	c.attempts[e.Agent] = *e.Activity
}

// Prefer the newest active peer; exhausted and completed attempts never dilute a live rate.
func (c *chatState) currentAttempt() (string, *runrecord.ModelActivity) {
	var selected string
	var latest runrecord.ModelActivity
	for agent, attempt := range c.attempts {
		if !latest.Started.IsZero() {
			if attempt.Finished != latest.Finished {
				if attempt.Finished {
					continue
				}
			} else if attempt.Started.Before(latest.Started) || attempt.Started.Equal(latest.Started) && agent >= selected {
				continue
			}
		}
		selected, latest = agent, attempt
	}
	if latest.Started.IsZero() {
		return "", nil
	}
	return selected, &latest
}
