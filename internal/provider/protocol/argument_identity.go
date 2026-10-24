// Purpose: Compare complete JSON argument values while preserving strings, numbers and unique keys.
// Pattern: Bounded pure parser at the Responses stream boundary.
// Usage: finalArguments accepts equivalent serialization without replacing the streamed source.
// Related: internal/provider/protocol/lmstudio_calls.go, internal/agent/tool/arguments.go.
package protocol

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func equivalentArguments(left, right string) bool {
	parse := func(text string) (any, error) {
		if len(text) > toolapi.MaxArgumentsBytes {
			return nil, fmt.Errorf("arguments exceed limit")
		}
		decoder := json.NewDecoder(strings.NewReader(text))
		decoder.UseNumber()
		value, err := argumentValue(decoder, 0)
		if err != nil {
			return nil, err
		}
		if _, ok := value.(map[string]any); !ok {
			return nil, fmt.Errorf("arguments must be an object")
		}
		if _, err = decoder.Token(); err != io.EOF {
			return nil, fmt.Errorf("trailing argument content")
		}
		return value, nil
	}
	a, err := parse(left)
	if err != nil {
		return false
	}
	b, err := parse(right)
	return err == nil && reflect.DeepEqual(a, b)
}

func argumentValue(d *json.Decoder, depth int) (any, error) {
	if depth > 64 {
		return nil, fmt.Errorf("argument nesting exceeds limit")
	}
	token, err := d.Token()
	if err != nil {
		return nil, err
	}
	switch token {
	case json.Delim('{'):
		fields := map[string]any{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return nil, err
			}
			name, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("invalid object key")
			}
			if _, exists := fields[name]; exists {
				return nil, fmt.Errorf("duplicate object key")
			}
			value, err := argumentValue(d, depth+1)
			if err != nil {
				return nil, err
			}
			fields[name] = value
		}
		end, err := d.Token()
		if err != nil || end != json.Delim('}') {
			return nil, fmt.Errorf("incomplete object")
		}
		return fields, nil
	case json.Delim('['):
		values := []any{}
		for d.More() {
			value, err := argumentValue(d, depth+1)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		end, err := d.Token()
		if err != nil || end != json.Delim(']') {
			return nil, fmt.Errorf("incomplete array")
		}
		return values, nil
	default:
		if _, ok := token.(json.Delim); ok {
			return nil, fmt.Errorf("unexpected argument delimiter")
		}
		return token, nil
	}
}
