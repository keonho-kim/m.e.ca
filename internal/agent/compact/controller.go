// Purpose: Compact a bounded projection of archived dialogue in one model call and atomically publish a validated memory revision.
// Pattern: Use Case.
// Usage: Refreshed only at a task decision boundary, never during native call assembly or approval.
// Related: internal/agent/compact/types.go, internal/agent/compact/context.go, internal/storage/sqlite/memory.go.
package compact

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
)

type Controller struct {
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
func (c *Controller) Refresh(ctx context.Context, generate Generate) (bool, error) {
	return c.refresh(ctx, generate, false)
}

// Compact responds to a confirmed context-capacity failure at a safe boundary.
func (c *Controller) Compact(ctx context.Context, generate Generate) (bool, error) {
	return c.refresh(ctx, generate, true)
}

func (c *Controller) refresh(ctx context.Context, generate Generate, force bool) (bool, error) {
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
	if !force && mark.Tokens < c.Settings.ThresholdTokens {
		return false, nil
	}
	boundary, err := c.Repository.CompactionBoundary(ctx, c.Session, mark.Timeline, c.Settings.KeepRecentBytes)
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
	base, err := c.summaryMessages(c.Redact(next.Markdown), nil, 0)
	if err != nil {
		return false, err
	}
	remaining := c.Settings.MaxInputBytes - 256
	for _, message := range base {
		remaining -= len(message.Content)
	}
	sources := sourceBuffer{limit: remaining}
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
	if len(sources.entries) == 0 {
		return false, nil
	}
	next.Markdown, err = c.summarize(ctx, generate, c.Redact(next.Markdown), sources.packet(), sources.omitted)
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
func (c *Controller) Context() (string, int64, error) {
	text, err := c.memory.Context()
	return text, c.memory.Through, err
}
