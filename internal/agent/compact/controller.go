// Purpose: Compact archived dialogue in bounded chronological chunks and atomically publish a validated memory revision.
// Pattern: Use Case.
// Usage: Refreshed only at a task decision boundary, never during native call assembly or approval.
// Related: internal/agent/compact/types.go, internal/agent/compact/ledger.go, internal/storage/sqlite/memory.go.
package compact

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	toolrecord "github.com/keonho-kim/v-code/internal/agent/tool"
)

type Controller struct {
	Inference              *inference.Settings
	InputTokens            int64
	Repository             Repository
	Session, Model, Prompt string
	Settings               Settings
	Redact                 func(string) string
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
	if !force && mark.Tokens < c.Settings.ThresholdTokens && (c.Settings.TargetInputTokens == 0 || c.InputTokens < c.Settings.TargetInputTokens) {
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
	next.Ledger.Reviews = append([]ReviewFact(nil), next.Ledger.Reviews...)
	next.Ledger.Groundings = append([]GroundingFact(nil), next.Ledger.Groundings...)
	next.Ledger.Runs = append([]RunFact(nil), next.Ledger.Runs...)
	next.Ledger.Operations = append([]toolrecord.Operation(nil), next.Ledger.Operations...)
	cursor := next.Through
	processed := false
	var batch []json.RawMessage
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		body, err := json.Marshal(batch)
		if err != nil {
			return err
		}
		summary, err := c.summarize(ctx, generate, next.Markdown, string(body))
		if err != nil {
			return err
		}
		next.Markdown = summary
		batch = nil
		processed = true
		return nil
	}
	for cursor < boundary {
		records, through, err := c.Repository.MemoryRecords(ctx, c.Session, cursor, boundary)
		if err != nil {
			return false, err
		}
		if through <= cursor {
			return false, fmt.Errorf("compaction source cursor did not advance")
		}
		for _, record := range records {
			next.Ledger.merge(record)

			// Segment the complete source representation, including large metadata fields.
			payload, err := json.Marshal(record)
			if err != nil {
				return false, err
			}
			parts := segments(string(payload), 1024)
			for index, part := range parts {
				source := struct {
					Sequence       int64
					Role, Fragment string
					Part, Parts    int
				}{record.Sequence, record.Role, part, index + 1, len(parts)}
				encoded, err := json.Marshal(source)
				if err != nil {
					return false, err
				}
				trial := append(append([]json.RawMessage(nil), batch...), encoded)
				body, err := json.Marshal(trial)
				if err != nil {
					return false, err
				}
				if _, err := c.summaryMessages(next.Markdown, string(body)); err != nil {
					if len(batch) == 0 {
						return false, err
					}
					if err := flush(); err != nil {
						return false, err
					}
					trial = []json.RawMessage{encoded}
					body, err = json.Marshal(trial)
					if err != nil {
						return false, err
					}
					if _, err := c.summaryMessages(next.Markdown, string(body)); err != nil {
						return false, err
					}
				}
				batch = trial
			}
		}
		cursor = through
	}
	if err := flush(); err != nil {
		return false, err
	}
	if !processed {
		return false, fmt.Errorf("token threshold reached without new source records; previous memory retained")
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
	c.InputTokens = 0
	return true, nil
}
func (c *Controller) Context() (string, int64, error) {
	text, err := c.memory.Context()
	return text, c.memory.Through, err
}
