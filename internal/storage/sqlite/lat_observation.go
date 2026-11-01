// Purpose: Project authenticated outcome excerpts without hydrating complete large output bodies.
// Pattern: Bounded repository projection over canonical payloads.
// Usage: LATRecords reads actual output and applied changes; user instructions use full hydration.
// Related: internal/storage/sqlite/lat_sources.go, internal/storage/content/excerpt.go.
package sqlite

import (
	"context"
	"fmt"

	json "github.com/goccy/go-json"

	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
)

func latObservation(ctx context.Context, record storedRecord, observed string, excerpts *reviewExcerpts) (string, error) {
	var fields map[string]json.RawMessage
	var refs map[string]string
	if err := json.Unmarshal(record.body, &fields); err != nil {
		return "", err
	}
	if err := json.Unmarshal(record.refs, &refs); err != nil {
		return "", err
	}
	var values [2]latmemory.ObservationExcerpt
	for i, field := range []string{"output", "change_preview"} {
		raw := fields[field]
		original := len(raw)
		key, _ := json.Marshal([]string{field})
		if id := refs[string(key)]; id != "" {
			if string(raw) != "null" {
				return "", fmt.Errorf("invalid review payload placeholder")
			}
			var err error
			raw, original, err = excerpts.excerpt(ctx, id)
			if err != nil {
				return "", err
			}
		}
		values[i] = latmemory.ObservationExcerpt{Body: raw, Original: original}
	}
	return latmemory.DescribeObservation(observed, values[0], values[1])
}
