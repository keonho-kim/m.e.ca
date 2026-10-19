// Purpose: Validate complete native tool argument objects against the declared parameter contract.
// Pattern: Pure parsing at the tool trust boundary.
// Usage: Runtime candidates and local execution validate the same schema before I/O.
// Related: internal/agent/tool/schema.go, internal/agent/tool/call.go, internal/tools/local/arguments.go.
package tool

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode/utf8"
)

func argumentObject(body json.RawMessage) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return nil, fmt.Errorf("arguments must be a JSON object")
	}
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("invalid argument key")
		}
		if _, exists := fields[key]; exists {
			return nil, fmt.Errorf("duplicate argument %s", key)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		fields[key] = value
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing tool arguments")
	}
	return fields, nil
}

func ValidateArguments(definition Definition, body json.RawMessage) (map[string]json.RawMessage, error) {
	if len(body) > MaxArgumentsBytes || !utf8.Valid(body) {
		return nil, fmt.Errorf("invalid or oversized arguments")
	}
	fields, err := argumentObject(body)
	if err != nil {
		return nil, err
	}
	for name := range fields {
		if !slices.ContainsFunc(definition.Parameters, func(p Parameter) bool { return p.Name == name }) {
			return nil, fmt.Errorf("unexpected %s argument: %s", definition.Name, name)
		}
	}
	for _, parameter := range definition.Parameters {
		value, present := fields[parameter.Name]
		if !present {
			if parameter.Required {
				return nil, fmt.Errorf("missing %s argument", parameter.Name)
			}
			continue
		}
		if err := validateValue(parameter, value); err != nil {
			return nil, err
		}
	}
	return fields, nil
}

func validateValue(p Parameter, value json.RawMessage) error {
	if string(value) == "null" {
		return fmt.Errorf("%s cannot be null", p.Name)
	}
	if p.Kind == ObjectList {
		var entries []json.RawMessage
		if err := json.Unmarshal(value, &entries); err != nil || len(entries) == 0 || len(entries) > p.Max {
			return fmt.Errorf("%s requires 1 to %d objects", p.Name, p.Max)
		}
		for i, entry := range entries {
			if _, err := ValidateArguments(Definition{Name: p.Name, Parameters: p.Fields}, entry); err != nil {
				return fmt.Errorf("%s[%d]: %w", p.Name, i, err)
			}
		}
		return nil
	}
	if p.Kind == TextList {
		var values []string
		if err := json.Unmarshal(value, &values); err != nil || len(values) < 1 || len(values) > p.Max {
			return fmt.Errorf("%s requires a bounded string array", p.Name)
		}
		for _, text := range values {
			if strings.TrimSpace(text) == "" || len(text) > p.MaxBytes || strings.ContainsRune(text, 0) || !utf8.ValidString(text) {
				return fmt.Errorf("invalid %s item", p.Name)
			}
		}
		return nil
	}
	var text string
	if err := json.Unmarshal(value, &text); err != nil || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return fmt.Errorf("%s requires a valid string", p.Name)
	}
	if len(text) > p.MaxBytes || !p.AllowEmpty && strings.TrimSpace(text) == "" {
		return fmt.Errorf("%s requires text within %d bytes", p.Name, p.MaxBytes)
	}
	if len(p.Values) > 0 && !slices.Contains(p.Values, text) {
		return fmt.Errorf("%s must be one of %s", p.Name, strings.Join(p.Values, ", "))
	}
	switch p.Kind {
	case Text:
	case LiteralPath:
		if strings.ContainsAny(text, "\r\n") {
			return fmt.Errorf("%s requires one literal path without line breaks", p.Name)
		}
	default:
		return fmt.Errorf("unsupported parameter kind")
	}
	return nil
}
