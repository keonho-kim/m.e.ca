// Purpose: Compact archived dialogue in bounded chronological chunks and atomically publish a validated memory revision.
// Pattern: Use Case.
// Usage: Refreshed only at a task decision boundary, never during argument collection or approval.
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
	if mark.Tokens < c.Settings.ThresholdTokens {
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
	next.Ledger.Goals = append([]GoalFact(nil), next.Ledger.Goals...)
	next.Ledger.Reviews = append([]ReviewFact(nil), next.Ledger.Reviews...)
	next.Ledger.Groundings = append([]GroundingFact(nil), next.Ledger.Groundings...)
	next.Ledger.Tasks = append([]TaskFact(nil), next.Ledger.Tasks...)
	next.Ledger.Operations = append([]toolrecord.Operation(nil), next.Ledger.Operations...)
	cursor := next.Through
	processed := false
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
			parts := segments(string(payload), 4096)
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
				summary, err := c.summarize(ctx, generate, next.Markdown, string(encoded))
				if err != nil {
					return false, err
				}
				next.Markdown = summary
				processed = true
			}
		}
		cursor = through
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
	return true, nil
}
func (c *Controller) Context() (string, int64, error) {
	text, err := c.memory.Context()
	return text, c.memory.Through, err
}
