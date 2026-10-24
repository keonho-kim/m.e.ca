// Purpose: Refit one summary packet once when exact provider tokenization rejects its input.
// Pattern: Bounded preparation recovery without chained summaries.
// Usage: Controller preserves prior memory until one complete summary is accepted.
// Related: internal/agent/compact/budget.go, internal/agent/compact/summarize.go, internal/agent/model/capacity.go.
package compact

import (
	"context"
	"errors"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func (c *Controller) summarizeSources(ctx context.Context, generate Generate, previous string, sources *sourceBuffer, budget summaryBudget) (string, error) {
	for attempt := 0; ; attempt++ {
		messages, err := c.summaryMessages(previous, sources.packet(), sources.omitted)
		if err != nil {
			return "", err
		}
		text, err := c.summarize(ctx, generate, messages, budget)
		if err == nil {
			return text, nil
		}
		var capacity *modelapi.ContextLimitError
		if attempt != 0 || !errors.As(err, &capacity) || capacity.Input <= 0 || capacity.Capacity <= 0 || ctx.Err() != nil {
			return "", err
		}
		packetBytes := 0
		for _, message := range messages {
			packetBytes += len(message.Content)
		}
		adjusted, fitErr := budget.fitObserved(packetBytes, capacity)
		if fitErr != nil {
			return "", errors.Join(err, fitErr)
		}
		sources, fitErr = sources.refit(adjusted.sourceBytes, c.Redact)
		if fitErr != nil {
			return "", errors.Join(err, fitErr)
		}
		budget = adjusted
		c.ContextWindow = min(c.contextWindow(), capacity.Capacity)
	}
}
