// Purpose: Compact a bounded projection of archived dialogue in one model call and atomically publish a validated memory revision.
// Pattern: Use Case.
// Usage: The runtime refreshes at decision boundaries and receives a start notification before preparation.
// Related: internal/agent/compact/types.go, internal/agent/compact/context.go, internal/storage/sqlite/memory.go.
package compact

import (
	"context"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/inference"
	"github.com/keonho-kim/meca/internal/agent/prompt"
)

type Controller struct {
	ContextWindow          int
	Inference              *inference.Settings
	Repository             Repository
	Session, Model, Prompt string
	Settings               Settings
	Redact                 func(string) string
	ActiveRequest          string
	memory                 Memory
	loaded                 bool
}

func (c *Controller) Observe(ctx context.Context, count TokenCount) error {
	return c.Repository.AddContextTokens(ctx, c.Session, count)
}
func (c *Controller) Refresh(ctx context.Context, started func() error, generate Generate) (bool, error) {
	return c.refresh(ctx, started, generate, false)
}

// Compact responds to a confirmed context-capacity failure at a safe boundary.
func (c *Controller) Compact(ctx context.Context, started func() error, generate Generate) (bool, error) {
	return c.refresh(ctx, started, generate, true)
}

func (c *Controller) refresh(ctx context.Context, started func() error, generate Generate, force bool) (bool, error) {
	if err := c.Settings.Check(); err != nil {
		return false, err
	}
	if c.Repository == nil || c.Redact == nil || generate == nil {
		return false, fmt.Errorf("compaction dependencies are required")
	}
	if err := prompt.CheckTemplate(c.Prompt); err != nil {
		return false, err
	}
	if !c.loaded {
		memory, err := c.Repository.Memory(ctx, c.Session)
		if err != nil {
			return false, err
		}
		if err = memory.Check(c.Settings.MaxSummaryBytes); err != nil {
			return false, err
		}
		c.memory = memory
		c.loaded = true
	}
	mark, err := c.Repository.MemoryWatermark(ctx, c.Session, c.memory.UsageThrough)
	if err != nil {
		return false, err
	}
	window := c.contextWindow()
	reserve := min(int(c.Settings.ReserveTokens), max(1, window/4))
	threshold := max(1, window-reserve)
	if !force && mark.Tokens < int64(threshold) {
		return false, nil
	}
	if started != nil {
		if err := started(); err != nil {
			return false, err
		}
	}
	boundary, err := c.Repository.CompactionBoundary(ctx, c.Session, mark.Timeline, min(c.Settings.KeepRecentTokens, window/4)*int(bytesPerToken))
	if err != nil {
		return false, err
	}
	if boundary < 0 || boundary > mark.Timeline {
		return false, fmt.Errorf("invalid compaction source boundary")
	}
	if boundary <= c.memory.Through {
		return false, nil
	}
	next := c.memory
	previous := c.Redact(next.Markdown)
	budget, err := c.budget(previous)
	if err != nil {
		return false, err
	}
	sources := sourceBuffer{limit: budget.sourceBytes}
	cursor := next.Through
	for cursor < boundary {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		records, through, err := c.Repository.MemoryRecords(ctx, c.Session, cursor, boundary)
		if err != nil {
			return false, err
		}
		if through <= cursor || through > boundary {
			return false, fmt.Errorf("invalid compaction source cursor")
		}
		for _, record := range records {
			if err := sources.add(record, c.Redact); err != nil {
				return false, err
			}
		}
		cursor = through
	}
	if sources.entries.Len() == 0 {
		return false, nil
	}
	next.Markdown, err = c.summarizeSources(ctx, generate, previous, &sources, budget)
	if err != nil {
		return false, err
	}
	next.Revision++
	next.Through = boundary
	next.UsageThrough = mark.Usage
	if err = next.Check(c.Settings.MaxSummaryBytes); err != nil {
		return false, err
	}
	if err = c.Repository.InstallMemory(ctx, c.Session, c.memory.Revision, next); err != nil {
		return false, err
	}
	c.memory = next
	return true, nil
}
func (c *Controller) Context() (string, int64) {
	if c.memory.Revision == 0 {
		return "", c.memory.Through
	}
	return c.memory.Markdown, c.memory.Through
}
