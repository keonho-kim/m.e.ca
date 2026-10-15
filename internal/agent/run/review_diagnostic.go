// Purpose: Persist masked internal review attempts without displaying model protocol text.
// Pattern: Journal adapter.
// Usage: Both review coordinators pass recordReviewAttempt to their review use cases.
// Related: internal/agent/reviewjson/object.go, internal/agent/run/monitor.go.
package run

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/reviewjson"
	"time"
)

func (s *execution) recordReviewAttempt(ctx context.Context, a reviewjson.Attempt) error {
	a.Raw = s.runner.Redact(a.Raw)
	a.Error = s.runner.Redact(a.Error)
	body, err := json.Marshal(a)
	if err != nil {
		return err
	}
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return s.record(recordCtx, Event{Kind: "review_attempt", Internal: true, Text: string(body)})
}
