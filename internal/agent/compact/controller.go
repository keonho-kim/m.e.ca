// Purpose: Compact a bounded projection of archived dialogue in one model call and atomically publish a validated memory revision.
// Pattern: Use Case.
// Usage: The runtime refreshes at decision boundaries and receives a start notification before preparation.
// Related: internal/agent/compact/types.go, internal/agent/compact/context.go, internal/storage/sqlite/memory.go.
package compact

import (
	"context"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"

	"github.com/keonho-kim/meca/internal/agent/inference"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
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

type compactionMode uint8

const (
	thresholdCompaction compactionMode = iota
	capacityCompaction
	manualCompaction
)

func (c *Controller) Observe(ctx context.Context, count TokenCount) error {
	return c.Repository.AddContextTokens(ctx, c.Session, count)
}

// Refresh checks the current input when tokenization is available and otherwise uses the saved estimate.
func (c *Controller) Refresh(ctx context.Context, budget modelapi.InputBudget, started func(modelapi.ContextMeasurement) error, generate Generate) (bool, error) {
	c.observeWindow(budget.Window)
	return c.refresh(ctx, started, generate, thresholdCompaction, modelapi.ContextMeasurement{Input: int64(budget.InputTokens), Source: "tokenizer", OutputBudget: budget.OutputTokens})
}

func (c *Controller) observeWindow(window int) {
	if window > 0 && (c.ContextWindow == 0 || window < c.ContextWindow) {
		c.ContextWindow = window
	}
}

// Compact responds to a confirmed context-capacity failure at a safe boundary.
func (c *Controller) Compact(ctx context.Context, failure modelapi.ContextLimitError, started func(modelapi.ContextMeasurement) error, generate Generate) (bool, error) {
	c.observeWindow(failure.Capacity)
	source := "provider"
	if failure.Tokenized {
		source = "tokenizer"
	}
	return c.refresh(ctx, started, generate, capacityCompaction, modelapi.ContextMeasurement{Input: int64(failure.Input), Source: source, OutputBudget: failure.RequestedOutput})
}

// CompactNow summarizes eligible history on an idle conversation's explicit request.
func (c *Controller) CompactNow(ctx context.Context, started func(modelapi.ContextMeasurement) error, generate Generate) (bool, error) {
	return c.refresh(ctx, started, generate, manualCompaction, modelapi.ContextMeasurement{})
}

// Load restores the last summary before the agent assembles a request.
func (c *Controller) Load(ctx context.Context) error {
	if c.loaded {
		return nil
	}
	if err := c.Settings.Check(); err != nil {
		return err
	}
	if c.Repository == nil || c.Redact == nil {
		return faults.Errorf(diagcatalog.CompactCompactionDependenciesAreRequired)
	}
	if err := prompt.CheckTemplate(c.Prompt); err != nil {
		return err
	}
	memory, err := c.Repository.Memory(ctx, c.Session)
	if err != nil {
		return err
	}
	if err := memory.Check(c.Settings.MaxSummaryBytes); err != nil {
		return err
	}
	c.memory, c.loaded = memory, true
	return nil
}

func (c *Controller) refresh(ctx context.Context, started func(modelapi.ContextMeasurement) error, generate Generate, mode compactionMode, input modelapi.ContextMeasurement) (bool, error) {
	if generate == nil {
		return false, faults.Errorf(diagcatalog.CompactCompactionDependenciesAreRequired)
	}
	if err := c.Load(ctx); err != nil {
		return false, err
	}
	window := c.contextWindow()
	// Use the first whole token count at or above 75 percent.
	threshold := window - window/4
	if mode == thresholdCompaction && input.Input > 0 && input.Input < int64(threshold) {
		return false, nil
	}
	mark, err := c.Repository.MemoryWatermark(ctx, c.Session, c.memory.UsageThrough)
	if err != nil {
		return false, err
	}
	if mode == thresholdCompaction && input.Input <= 0 && mark.Tokens < int64(threshold) {
		return false, nil
	}
	keep := 0
	if mode != manualCompaction {
		keep = min(c.Settings.KeepRecentTokens, window/4)
		if rejectedOutput := input.OutputBudget; mode == capacityCompaction && rejectedOutput > 0 {
			keep = min(keep, max(1, (window-rejectedOutput)/2))
		}
	}
	boundary, err := c.Repository.CompactionBoundary(ctx, c.Session, mark.Timeline, keep*int(bytesPerToken))
	// A rejected request can be shorter than the preferred recent tail in bytes.
	// Keep the newest complete group, then compact only the eligible prefix.
	if err == nil && mode == capacityCompaction && boundary <= c.memory.Through && keep > 0 {
		boundary, err = c.Repository.CompactionBoundary(ctx, c.Session, mark.Timeline, 1)
	}
	if err != nil {
		return false, err
	}
	if boundary < 0 || boundary > mark.Timeline {
		return false, faults.Errorf(diagcatalog.CompactInvalidCompactionSourceBoundary)
	}
	if boundary <= c.memory.Through {
		return false, nil
	}
	if started != nil {
		input.Window = window
		if input.Input <= 0 {
			input.Input, input.Source = mark.Tokens, "watermark"
			if mark.Estimated {
				input.Source = "watermark_estimate"
			}
		}
		if err := started(input); err != nil {
			return false, err
		}
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
			return false, faults.Errorf(diagcatalog.CompactInvalidCompactionSourceCursor)
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
