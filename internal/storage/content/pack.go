// Purpose: Separate large JSON values into content-addressed references before persistence.
// Pattern: Pure record transformation with a caller-owned payload write callback.
// Usage: SQLite packs a record and writes referenced payloads in the same transaction.
// Related: internal/storage/content/restore.go, internal/storage/sqlite/record_codec.go.
package content

import (
	"strconv"

	json "github.com/goccy/go-json"
)

const (
	Threshold = 32 * 1024
	MaxBytes  = 512 * 1024 * 1024
)

func Pack(body []byte, put func([]byte) (string, error)) ([]byte, []byte, error) {
	if len(body) <= Threshold {
		return body, []byte("{}"), nil
	}
	refs := map[string]string{}
	var pack func(json.RawMessage, []string) (json.RawMessage, error)
	pack = func(raw json.RawMessage, path []string) (json.RawMessage, error) {
		if len(raw) <= Threshold {
			return raw, nil
		}
		opaque := len(path) > 0 && (path[len(path)-1] == "arguments" || path[len(path)-1] == "output" || path[len(path)-1] == "capture")
		if raw[0] == '"' || opaque {
			id, err := put(raw)
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
