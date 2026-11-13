// Purpose: Select one tool field without copying or decoding unrelated values.
// Pattern: Bounded top-level JSON projection over validated immutable bytes.
// Usage: Detail requests list small field keys or page a selected raw value.
// Related: internal/serverapi/detail/view.go, internal/commands/server/detail.go.
package detail

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func selectToolField(ctx context.Context, value any, field string, list bool) (any, error) {
	var raw []byte
	switch v := value.(type) {
	case json.RawMessage:
		raw = v
	case []byte:
		raw = v
	default:
		if list {
			return []string{}, nil
		}
		return nil, fault.New(diag.NotFound, diag.ServerToolDetailWasNotFound)
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) < 2 || raw[0] != '{' || !json.Valid(raw) {
		if list {
			return []string{}, nil
		}
		return nil, fault.New(diag.NotFound, diag.ServerToolDetailWasNotFound)
	}
	keys := make([]string, 0, 16)
	var selected json.RawMessage
	seen := make(map[string]bool)
	keyBytes := 0
	space := func(at int) int {
		for at < len(raw) && (raw[at] == ' ' || raw[at] == '\n' || raw[at] == '\r' || raw[at] == '\t') {
			at++
		}
		return at
	}
	for at := space(1); at < len(raw)-1; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := quotedEnd(raw, at)
		var key string
		if end-at <= 2048 {
			_ = json.Unmarshal(raw[at:end], &key)
		}
		start := space(space(end) + 1)
		end = jsonValueEnd(raw, start)
		if !list && key == field {
			if selected != nil {
				return nil, fault.New(diag.InvalidArgument, diag.ServerPageInvalid)
			}
			selected = raw[start:end]
		}
		if list {
			if key == "" || len(key) > 512 || seen[key] {
				return []string{}, nil
			}
			seen[key] = true
			keys = append(keys, key)
			keyBytes += len(key)
		}
		if list && (len(keys) > 64 || keyBytes > 4096) {
			return []string{}, nil
		}
		at = space(end)
		if raw[at] == '}' {
			break
		}
		at = space(at + 1)
	}
	if !list {
		if selected != nil {
			return selected, nil
		}
		return nil, fault.New(diag.NotFound, diag.ServerToolDetailWasNotFound)
	}
	slices.Sort(keys)
	return keys, nil
}

func quotedEnd(raw []byte, start int) int {
	for at := start + 1; at < len(raw); at++ {
		if raw[at] == '\\' {
			at++
			continue
		}
		if raw[at] == '"' {
			return at + 1
		}
	}
	return len(raw)
}

func jsonValueEnd(raw []byte, start int) int {
	depth := 0
	for at := start; at < len(raw); at++ {
		switch raw[at] {
		case '"':
			at = quotedEnd(raw, at) - 1
			if depth == 0 {
				return at + 1
			}
		case '{', '[':
			depth++
		case '}', ']':
			if depth == 0 {
				return at
			}
			depth--
			if depth == 0 {
				return at + 1
			}
		case ',':
			if depth == 0 {
				return at
			}
		}
	}
	return len(raw)
}
