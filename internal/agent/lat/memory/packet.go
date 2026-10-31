// Purpose: Select chronological original records within a model-sized summary request.
// Pattern: Pure packet builder with explicit covered boundaries.
// Usage: Controller plans one prefix; newer records remain outside its committed boundary.
// Related: internal/agent/lat/memory/controller.go, internal/agent/prompt/defaults/lat_compaction.md.
package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
)

type Plan struct {
	Request modelapi.Request
	Through int64
}

func inputBudget(instructions string, window, output int) (int, int, error) {
	if err := prompt.CheckTemplate(instructions); err != nil {
		return 0, 0, err
	}
	if window <= 0 {
		window = 262144
	}
	if output == 0 {
		output = 2048
	}
	output = min(output, window/4)
	budget := min(MaxBytes, window-output-len(instructions)-1024)
	if output < 1 || budget < 1024 {
		return 0, 0, fmt.Errorf("LAT memory instructions leave insufficient model context")
	}
	return budget, output, nil
}

func (c *Controller) records(ctx context.Context, after, through int64, budget int, current string, latestUser int64) ([]json.RawMessage, int64, error) {
	var selected []json.RawMessage
	used, cursor := 2, after
	for cursor < through {
		page, next, err := c.Repository.LATRecords(ctx, c.Session, cursor, through, budget-used)
		if errors.Is(err, ErrSourceBudget) && current != "" && len(current) > budget-used && len(current) <= MaxBytes {
			// The exact current occurrence is supplied outside this packet. Allow
			// its accepted original through the repository boundary to compare and
			// exclude it before enforcing the smaller remaining evidence budget.
			page, next, err = c.Repository.LATRecords(ctx, c.Session, cursor, through, MaxBytes)
		}
		if err != nil {
			if errors.Is(err, ErrSourceBudget) && len(selected) > 0 {
				return selected, cursor, nil
			}
			return nil, after, err
		}
		if next <= cursor || next > through {
			return nil, after, fmt.Errorf("invalid LAT source boundary")
		}
		for _, record := range page {
			record.Text = c.Redact(record.Text)
			if record.Kind == "user" && record.Sequence == latestUser && record.Text == current {
				cursor = record.Sequence
				continue
			}
			if len(record.Text) > budget-used {
				if len(selected) == 0 {
					return nil, after, fmt.Errorf("LAT original %s record cannot fit available context; original retained", record.Kind)
				}
				return selected, cursor, nil
			}
			body, err := json.Marshal(record)
			if err != nil {
				return nil, after, err
			}
			if used+len(body)+1 > budget {
				if len(selected) == 0 {
					return nil, after, fmt.Errorf("LAT original %s record cannot fit the available context; original retained", record.Kind)
				}
				return selected, cursor, nil
			}
			selected = append(selected, body)
			used += len(body) + 1
			cursor = record.Sequence
		}
		cursor = next
	}
	return selected, cursor, nil
}

func (c *Controller) plan(ctx context.Context, state State, mark Mark, instructions string, window, output int) (Plan, error) {
	budget, output, err := inputBudget(instructions, window, output)
	if err != nil {
		return Plan{}, err
	}
	packet := struct {
		Previous string            `json:"earlier_review_memory"`
		Sources  []json.RawMessage `json:"original_records_oldest_first"`
	}{Previous: c.Redact(state.Text)}
	fixed, err := json.Marshal(packet)
	if err != nil || len(fixed)+512 > budget {
		return Plan{}, fmt.Errorf("LAT prior memory cannot fit the available model context; previous memory retained")
	}
	packet.Sources, mark.Through, err = c.records(ctx, state.Through, mark.Through, budget-len(fixed)-512, "", 0)
	if err != nil {
		return Plan{}, err
	}
	if len(packet.Sources) == 0 {
		return Plan{}, nil
	}
	body, err := json.Marshal(packet)
	if err != nil || len(body) > budget {
		return Plan{}, fmt.Errorf("LAT memory request exceeds its model context")
	}
	return Plan{Through: mark.Through, Request: modelapi.Request{MaxOutput: output, Messages: []modelapi.Message{{Role: "system", Content: instructions}, {Role: "user", Content: string(body)}}}}, nil
}
