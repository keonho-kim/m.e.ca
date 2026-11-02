// Purpose: Restore selected JSON fields from content references without depending on a database.
// Pattern: Record transformation with a caller-owned payload read callback.
// Usage: SQLite releases its result rows before resolving record references.
// Related: internal/storage/content/pack.go, internal/storage/sqlite/record_codec.go.
package content

import (
	"bytes"
	"fmt"
	"strconv"

	json "github.com/goccy/go-json"
)

func Restore(body, refsBody []byte, fields map[string]bool, load func(string) (json.RawMessage, error)) ([]byte, error) {
	if fields == nil && bytes.Equal(refsBody, []byte("{}")) {
		if !json.Valid(body) {
			return nil, fmt.Errorf("invalid stored record JSON")
		}
		return body, nil
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	if fields != nil {
		for key := range root {
			if !fields[key] {
				delete(root, key)
			}
		}
	}
	var refs map[string]string
	if err := json.Unmarshal(refsBody, &refs); err != nil {
		return nil, err
	}
	loaded := map[string]json.RawMessage{}
	for key, id := range refs {
		var path []string
		if err := json.Unmarshal([]byte(key), &path); err != nil || len(path) == 0 {
			return nil, fmt.Errorf("invalid stored payload path")
		}
		if fields != nil && !fields[path[0]] {
			continue
		}
		value := loaded[id]
		if value == nil {
			var err error
			value, err = load(id)
			if err != nil {
				return nil, err
			}
			loaded[id] = value
		}
		raw, ok := root[path[0]]
		if !ok {
			return nil, fmt.Errorf("missing stored payload field")
		}
		raw, err := restoreValue(raw, path[1:], value)
		if err != nil {
			return nil, err
		}
		root[path[0]] = raw
	}
	return json.Marshal(root)
}

func restoreValue(raw json.RawMessage, path []string, value json.RawMessage) (json.RawMessage, error) {
	if len(path) == 0 {
		if string(raw) != "null" {
			return nil, fmt.Errorf("invalid stored payload placeholder")
		}
		return value, nil
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("invalid stored payload path")
	}
	if raw[0] == '{' {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return nil, err
		}
		child, ok := fields[path[0]]
		if !ok {
			return nil, fmt.Errorf("missing stored payload field")
		}
		next, err := restoreValue(child, path[1:], value)
		if err != nil {
			return nil, err
		}
		fields[path[0]] = next
		return json.Marshal(fields)
	}
	if raw[0] == '[' {
		var values []json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
		i, err := strconv.Atoi(path[0])
		if err != nil || i < 0 || i >= len(values) {
			return nil, fmt.Errorf("invalid stored payload index")
		}
		next, err := restoreValue(values[i], path[1:], value)
		if err != nil {
			return nil, err
		}
		values[i] = next
		return json.Marshal(values)
	}
	return nil, fmt.Errorf("invalid stored payload container")
}
