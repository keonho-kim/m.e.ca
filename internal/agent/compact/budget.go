// Purpose: Reserve summary output and fixed instructions before selecting older conversation sources.
// Pattern: Pure context-budget calculation.
// Usage: Controller uses one budget for source selection and the final summary request.
// Related: internal/agent/compact/controller.go, internal/agent/compact/summarize.go, internal/agent/compact/usage.go.
package compact

import (
	"fmt"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type summaryBudget struct {
	inputBytes, sourceBytes, outputTokens int
}

func (c *Controller) contextWindow() int {
	if c.ContextWindow == 0 {
		return 262144
	}
	return c.ContextWindow
}

func (c *Controller) budget(previous string) (summaryBudget, error) {
	parameters, err := inference.Resolve(c.Inference, inference.Compaction)
	if err != nil {
		return summaryBudget{}, err
	}
	window := c.contextWindow()
	output, err := modelapi.OutputBudget(parameters.MaxOutputTokens, window, 0, 0)
	if err != nil {
		return summaryBudget{}, err
	}
	// This is a source-selection estimate, never reported as provider token usage.
	input := int(min(int64(c.Settings.MaxInputBytes), int64(window-output)*bytesPerToken))
	base, err := c.summaryMessages(previous, nil, 0)
	if err != nil {
		return summaryBudget{}, err
	}
	// Allow room for chat framing and changes to the source envelope/count.
	remaining := input - 256
	for _, message := range base {
		remaining -= len(message.Content)
	}
	if remaining <= 0 {
		return summaryBudget{}, fmt.Errorf("compaction instructions, current request and previous memory exceed the %d-byte summary input budget at %d context tokens with %d output tokens reserved; original history retained; increase model context or adjust compaction input/output limits", input, window, output)
	}
	return summaryBudget{inputBytes: input, sourceBytes: remaining, outputTokens: output}, nil
}

func (b summaryBudget) fitObserved(packetBytes int, capacity *modelapi.ContextLimitError) (summaryBudget, error) {
	output := min(b.outputTokens, capacity.Capacity-1)
	// Actual input occupancy provides a packet-specific ratio. Leave extra
	// room for template framing and changed source selection density.
	available := capacity.Capacity - output
	input := min(b.inputBytes-1, int(int64(packetBytes)*int64(available)/int64(capacity.Input)*9/10)-256)
	remaining := b.sourceBytes - (b.inputBytes - input)
	if remaining <= 0 {
		return summaryBudget{}, fmt.Errorf("compaction fixed input cannot fit the measured model context with output reserved; original history retained")
	}
	return summaryBudget{inputBytes: input, sourceBytes: remaining, outputTokens: output}, nil
}
