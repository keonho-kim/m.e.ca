// Purpose: Aggregate actual usage events and weighted generation time without retaining event bodies.
// Pattern: Small value reducer.
// Usage: Team runtime and terminal counters fold the same completed usage observations.
// Related: internal/agent/record/statistics.go, internal/tui/session/team_events.go.
package record

import modelapi "github.com/keonho-kim/meca/internal/agent/model"

type UsageTotals struct {
	Usage          modelapi.TokenUsage
	MeasuredMillis int64
	Received       bool
}

func (t *UsageTotals) Add(event Event) {
	if event.Usage == nil {
		return
	}
	t.Received = true
	t.Usage = t.Usage.Add(*event.Usage)
	if event.Generation != nil {
		t.MeasuredMillis += event.Generation.MeasuredMillis
	}
}
func (t UsageTotals) Snapshot(elapsed int64, concurrent bool) Statistics {
	result := Statistics{ElapsedMillis: elapsed, Concurrent: concurrent}
	if t.Received {
		usage := t.Usage.Clone()
		result.Usage = &usage
		result.GenerationMillis = t.MeasuredMillis
	}
	return result
}
