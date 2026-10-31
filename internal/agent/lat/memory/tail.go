// Purpose: Assemble a complete independent memo plus every later original observation.
// Pattern: Bounded context value with explicit compaction demand.
// Usage: Final review retries after a summary only when the original tail does not fit.
// Related: internal/agent/lat/memory/controller.go, internal/agent/lat/packet.go.
package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

var ErrNeedsCompaction = errors.New("independent review source tail needs compaction")

func (c *Controller) tail(ctx context.Context, state State, mark Mark, budget int, current string) ([]byte, error) {
	packet := struct {
		Memory string            `json:"independent_review_memory"`
		Recent []json.RawMessage `json:"later_original_records_oldest_first"`
	}{Memory: c.Redact(state.Text)}
	fixed, err := json.Marshal(packet)
	if err != nil || len(fixed)+512 >= budget {
		return nil, fmt.Errorf("independent review memo cannot fit the review context")
	}
	var through int64
	packet.Recent, through, err = c.records(ctx, state.Through, mark.Through, budget-len(fixed)-512, current, mark.LatestUser)
	if err != nil {
		return nil, errors.Join(ErrNeedsCompaction, err)
	}
	if through < mark.Through {
		return nil, ErrNeedsCompaction
	}
	body, err := json.Marshal(packet)
	if err != nil || len(body) > budget {
		return nil, ErrNeedsCompaction
	}
	return body, nil
}
