// Purpose: Separate large JSON values into an envelope and caller-resolved references.
// Pattern: Pure record transformation with a caller-owned payload write callback.
// Usage: SQLite packs a record and writes referenced payloads in the same transaction.
// Related: internal/storage/content/restore.go, internal/storage/sqlite/record_codec.go.
package content

import (
	"strconv"

	json "github.com/goccy/go-json"
)

const (
	Threshold       = 32 * 1024
	SharedThreshold = 512
	MaxBytes        = 512 * 1024 * 1024
)

func Pack(body []byte, sharedField string, put func([]byte) (string, error)) ([]byte, map[string]string, error) {
	minimum := Threshold
	if sharedField != "" {
		minimum = SharedThreshold
	}
	if len(body) <= minimum {
		return body, nil, nil
	}
	refs := map[string]string{}
	var pack func(json.RawMessage, []string) (json.RawMessage, error)
	pack = func(raw json.RawMessage, path []string) (json.RawMessage, error) {
		limit := Threshold
		if len(path) == 1 && path[0] == sharedField {
			limit = SharedThreshold
		}
		if (len(path) > 0 && len(raw) <= limit) || (len(path) == 0 && len(raw) <= minimum) {
			return raw, nil
		}
		opaque := len(path) > 0 && (path[len(path)-1] == "arguments" || path[len(path)-1] == "output" || path[len(path)-1] == "capture")
		if raw[0] == '"' || opaque {
			payload := []byte(raw)
			quote := false
			if raw[0] == '"' {
				var text string
				if json.Unmarshal(raw, &text) == nil && json.Valid([]byte(text)) {
					payload = []byte(text)
					quote = true
				}
			}
			id, err := put(payload)
			if err != nil {
				return nil, err
			}
			key, _ := json.Marshal(path)
			if quote {
				id = StringReference(id)
			}
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
	return packed, refs, nil
}

func appendPath(path []string, key string) []string {
	next := make([]string, len(path)+1)
	copy(next, path)
	next[len(path)] = key
	return next
}
