// Purpose: Assemble a complete independent memo plus every later original observation.
// Pattern: Bounded context value with explicit compaction demand.
// Usage: Final review retries after a summary only when the original tail does not fit.
// Related: internal/agent/lat/memory/controller.go, internal/agent/lat/packet.go.
package memory

import (
	"context"
	"errors"
	"fmt"
)

var ErrNeedsCompaction = errors.New("independent review source tail needs compaction")

func (c *Controller) tail(ctx context.Context, state State, mark Mark, budget int, current string) ([]byte, error) {
	previous := c.Redact(state.Text)
	fixed, err := sourcePacket("independent_review_memory", "later_original_records_oldest_first", previous, nil)
	if err != nil || len(fixed)+512 >= budget {
		return nil, fmt.Errorf("independent review memo cannot fit the review context")
	}
	recent, through, err := c.records(ctx, state.Through, mark.Through, budget-len(fixed)-512, current, mark.LatestUser)
	if err != nil {
		return nil, errors.Join(ErrNeedsCompaction, err)
	}
	if through < mark.Through {
		return nil, ErrNeedsCompaction
	}
	body, err := sourcePacket("independent_review_memory", "later_original_records_oldest_first", previous, recent)
	if err != nil || len(body) > budget {
		return nil, ErrNeedsCompaction
	}
	return body, nil
}
