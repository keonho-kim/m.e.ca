// Purpose: Project authenticated outcome excerpts without hydrating complete large output bodies.
// Pattern: Bounded repository projection over canonical payloads.
// Usage: LATRecords reads actual output and applied changes; user instructions use full hydration.
// Related: internal/storage/sqlite/lat_sources.go, internal/storage/content/excerpt.go.
package sqlite

import (
	"context"
	"fmt"
	"unicode/utf8"

	json "github.com/goccy/go-json"
	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
)

func (s *Store) latObservation(ctx context.Context, record storedRecord, observed string) (string, error) {
	var fields map[string]json.RawMessage
	var refs map[string]string
	if err := json.Unmarshal(record.body, &fields); err != nil {
		return "", err
	}
	if err := json.Unmarshal(record.refs, &refs); err != nil {
		return "", err
	}
	values := map[string]string{}
	for _, field := range []string{"output", "change_preview"} {
		raw := fields[field]
		original := len(raw)
		key, _ := json.Marshal([]string{field})
		if id := refs[string(key)]; id != "" {
			if string(raw) != "null" {
				return "", fmt.Errorf("invalid review payload placeholder")
			}
			var codec int
			var body []byte
			if err := s.db.QueryRowContext(ctx, "SELECT codec,raw_bytes,body FROM payloads WHERE id=?", id).Scan(&codec, &original, &body); err != nil {
				return "", err
			}
			var err error
			raw, err = s.codec.Excerpt(ctx, codec, original, id, body, 8192)
			if err != nil {
				return "", err
			}
		}
		if len(raw) < original {
			for trim := 0; trim < 3 && !utf8.Valid(raw); trim++ {
				raw = raw[:len(raw)-1]
			}
		}
		value := string(raw)
		if field == "change_preview" && len(raw) > 0 {
			// Stored diffs are JSON strings. Complete a clipped string only at a
			// valid escape boundary; the complete source was authenticated above.
			if len(raw) < original {
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
		if len(raw) < original {
			value += "\n[excerpt; original record retained]"
		}
		values[field] = latmemory.Excerpt(value, 8192)
	}
	body, err := json.Marshal(struct {
		Observed string `json:"observed_outcome"`
		Output   string `json:"output_excerpt,omitempty"`
		Diff     string `json:"applied_diff_excerpt,omitempty"`
	}{latmemory.Excerpt(observed, latmemory.TargetBytes), values["output"], values["change_preview"]})
	return string(body), err
}
