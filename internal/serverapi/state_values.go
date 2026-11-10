// Purpose: Convert runtime time and usage values into public state fields.
// Pattern: Pure wire projection.
// Usage: Snapshot execution entries use these shared conversions.
// Related: internal/serverapi/projection.go, api/meca/v1/state.proto.
package serverapi

import (
	"github.com/keonho-kim/meca/internal/agent/record"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"time"
)

func wireTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
func wireStatistics(s record.Statistics) *v1.Statistics {
	value := &v1.Statistics{Concurrent: s.Concurrent, ElapsedMillis: s.ElapsedMillis, GenerationMillis: s.GenerationMillis}
	if u := s.Usage; u != nil {
		value.Usage = &v1.Usage{Input: u.Input, Output: u.Output, Cost: u.Cost}
	}
	return value
}
