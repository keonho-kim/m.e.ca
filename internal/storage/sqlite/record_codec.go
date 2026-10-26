// Purpose: Replace large JSON leaf values with private payload references and restore them for public reads.
// Pattern: Session record codec.
// Usage: Message/event writes pack records; query adapters hydrate only the fields their contract requires.
// Related: internal/storage/sqlite/payload.go, internal/storage/sqlite/record_page.go.
package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
)

type storedRecord struct {
	sequence   int64
	kind       string
	body, refs []byte
	size       int
}

func (s *Store) packRecord(ctx context.Context, tx *sql.Tx, body []byte) ([]byte, []byte, error) {
	if len(body) <= payloadThreshold {
		return body, []byte("{}"), nil
	}
	refs := map[string]string{}
	var pack func(json.RawMessage, []string) (json.RawMessage, error)
	pack = func(raw json.RawMessage, path []string) (json.RawMessage, error) {
		if len(raw) <= payloadThreshold {
			return raw, nil
		}
		opaque := len(path) > 0 && (path[len(path)-1] == "arguments" || path[len(path)-1] == "output" || path[len(path)-1] == "capture")
		if raw[0] == '"' || opaque {
			id, err := s.putPayload(ctx, tx, raw)
			if err != nil {
				return nil, err
			}
			key, _ := json.Marshal(path)
			refs[string(key)] = id
			return json.RawMessage("null"), nil
		}
		switch raw[0] {
		case '{':
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(raw, &fields); err != nil {
				return nil, err
			}
			for key, value := range fields {
				next, err := pack(value, appendPath(path, key))
				if err != nil {
					return nil, err
				}
				fields[key] = next
			}
			return json.Marshal(fields)
		case '[':
			var values []json.RawMessage
			if err := json.Unmarshal(raw, &values); err != nil {
				return nil, err
			}
			for i, value := range values {
				next, err := pack(value, appendPath(path, strconv.Itoa(i)))
				if err != nil {
					return nil, err
				}
				values[i] = next
			}
			return json.Marshal(values)
		}
		return raw, nil
	}
	packed, err := pack(body, nil)
	if err != nil {
		return nil, nil, err
	}
	encoded, err := json.Marshal(refs)
	return packed, encoded, err
}

func appendPath(path []string, key string) []string {
	next := make([]string, len(path)+1)
	copy(next, path)
	next[len(path)] = key
	return next
}

// Hydration happens after the result rows close: this DB owns one connection.
func (s *Store) unpackRecord(ctx context.Context, record storedRecord, fields map[string]bool) ([]byte, error) {
	if fields == nil && bytes.Equal(record.refs, []byte("{}")) {
		if !json.Valid(record.body) {
			return nil, fmt.Errorf("invalid stored record JSON")
		}
		return record.body, nil
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(record.body, &root); err != nil {
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
	if err := json.Unmarshal(record.refs, &refs); err != nil {
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
			value, err = s.payload(ctx, id)
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
