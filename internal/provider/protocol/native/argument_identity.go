// Purpose: Compare complete JSON argument values while preserving strings, numbers and unique keys.
// Pattern: Bounded pure parser at the Responses stream boundary.
// Usage: Final and signed Gemini history accept equivalent serialization without replacing the source.
// Related: internal/provider/protocol/responses/calls.go, internal/agent/tool/arguments.go.
package native

import (
	"encoding/json"

	"io"
	"reflect"
	"strings"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func SameArguments(left, right string) bool {
	parse := func(text string) (any, error) {
		if len(text) > toolapi.MaxArgumentsBytes {
			return nil, faults.Errorf(diagcatalog.NativeArgumentsExceedLimit)
		}
		decoder := json.NewDecoder(strings.NewReader(text))
		decoder.UseNumber()
		value, err := argumentValue(decoder, 0)
		if err != nil {
			return nil, err
		}
		if _, ok := value.(map[string]any); !ok {
			return nil, faults.Errorf(diagcatalog.NativeArgumentsMustBeAnObject)
		}
		if _, err = decoder.Token(); err != io.EOF {
			return nil, faults.Errorf(diagcatalog.NativeTrailingArgumentContent)
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
		return nil, faults.Errorf(diagcatalog.NativeArgumentNestingExceedsLimit)
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
				return nil, faults.Errorf(diagcatalog.NativeInvalidObjectKey)
			}
			if _, exists := fields[name]; exists {
				return nil, faults.Errorf(diagcatalog.NativeDuplicateObjectKey)
			}
			value, err := argumentValue(d, depth+1)
			if err != nil {
				return nil, err
			}
			fields[name] = value
		}
		end, err := d.Token()
		if err != nil || end != json.Delim('}') {
			return nil, faults.Errorf(diagcatalog.NativeIncompleteObject)
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
			return nil, faults.Errorf(diagcatalog.NativeIncompleteArray)
		}
		return values, nil
	default:
		if _, ok := token.(json.Delim); ok {
			return nil, faults.Errorf(diagcatalog.NativeUnexpectedArgumentDelimiter)
		}
		return token, nil
	}
}
