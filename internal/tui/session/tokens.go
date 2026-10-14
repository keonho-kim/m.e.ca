// Purpose: Project final conversation usage and the active call without double-counting stream snapshots.
// Pattern: Presentation projection.
// Usage: Menu, chat and browser renderers read displayTokens on the terminal event loop.
// Related: internal/tui/session/events.go, internal/tui/view/tokens.go.
package session

import modelapi "github.com/keonho-kim/v-code/internal/agent/model"

func (a *app) displayTokens() modelapi.TokenUsage {
	return modelapi.TokenUsage{
		Input:  a.tokenTotal.Input + a.tokenCurrent.Input,
		Output: a.tokenTotal.Output + a.tokenCurrent.Output,
	}
}
