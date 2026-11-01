// Purpose: Select nested display fields before restoring referenced payloads.
// Pattern: Pure structural projection.
// Usage: Restore uses Selection only for nested objects that need narrower display fields.
// Related: internal/storage/content/restore.go, internal/storage/sqlite/timeline.go.
package content

import (
	"fmt"

	json "github.com/goccy/go-json"
)

// Selection keeps named fields; nil retains the complete value and * selects array elements.
type Selection map[string]Selection

func (s Selection) allows(path []string) bool {
	for _, part := range path {
		if s == nil {
			return true
		}
		next, ok := s[part]
		if !ok {
			next, ok = s["*"]
		}
		if !ok {
			return false
		}
		s = next
	}
	return true
}

func projectValue(raw json.RawMessage, fields Selection) (json.RawMessage, error) {
	if fields == nil || string(raw) == "null" {
		return raw, nil
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("missing projected value")
	}
	if raw[0] == '[' {
		var values []json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
		child, ok := fields["*"]
		if !ok {
			return nil, fmt.Errorf("array projection requires an element selection")
		}
		for i := range values {
			var err error
			values[i], err = projectValue(values[i], child)
			if err != nil {
				return nil, err
			}
		}
		return json.Marshal(values)
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	for key, value := range values {
		child, ok := fields[key]
		if !ok {
			delete(values, key)
			continue
		}
		projected, err := projectValue(value, child)
		if err != nil {
			return nil, err
		}
		values[key] = projected
	}
	return json.Marshal(values)
}
