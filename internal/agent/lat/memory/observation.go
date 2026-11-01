// Purpose: Format authenticated outcome excerpts as compact independent-review evidence.
// Pattern: Pure observation projection.
// Usage: Storage supplies original lengths and already-authenticated prefixes without SQL details.
// Related: internal/storage/sqlite/lat_observation.go, internal/agent/lat/memory/types.go.
package memory

import (
	"unicode/utf8"

	json "github.com/goccy/go-json"
)

// ObservationExcerpt distinguishes a complete value from an authenticated prefix.
type ObservationExcerpt struct {
	Body     []byte
	Original int
}

func DescribeObservation(observed string, output, change ObservationExcerpt) (string, error) {
	out, err := observationText(output, false)
	if err != nil {
		return "", err
	}
	diff, err := observationText(change, true)
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(struct {
		Observed string `json:"observed_outcome"`
		Output   string `json:"output_excerpt,omitempty"`
		Diff     string `json:"applied_diff_excerpt,omitempty"`
	}{Excerpt(observed, TargetBytes), out, diff})
	return string(body), err
}

func observationText(source ObservationExcerpt, quoted bool) (string, error) {
	raw := source.Body
	if len(raw) < source.Original {
		for trim := 0; trim < 3 && !utf8.Valid(raw); trim++ {
			raw = raw[:len(raw)-1]
		}
	}
	value := string(raw)
	if quoted && len(raw) > 0 {
		if len(raw) < source.Original {
			// Reconstruct only a valid JSON-string prefix; the storage adapter
			// authenticated the complete source before supplying this excerpt.
			for trim := 0; trim <= 6 && trim < len(raw); trim++ {
				prefix := raw[:len(raw)-trim]
				if utf8.Valid(prefix) && json.Unmarshal(append(append([]byte(nil), prefix...), '"'), &value) == nil {
					break
				}
			}
		} else if err := json.Unmarshal(raw, &value); err != nil {
			return "", err
		}
	}
	if len(raw) < source.Original {
		value += "\n[excerpt; original record retained]"
	}
	return Excerpt(value, 8192), nil
}
