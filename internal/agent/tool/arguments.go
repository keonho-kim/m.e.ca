// Purpose: Validate complete native tool argument objects against the declared parameter contract.
// Pattern: Pure parsing at the tool trust boundary.
// Usage: Runtime candidates and local execution validate the same schema before I/O.
// Related: internal/agent/tool/schema.go, internal/agent/tool/call.go, internal/tools/local/catalog/arguments.go.
package tool

import (
	"bytes"
	"encoding/json"

	"io"
	"slices"
	"strings"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func argumentObject(body json.RawMessage) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return nil, faults.Errorf(diagcatalog.ToolArgumentsMustBeAJSONObject)
	}
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, faults.Errorf(diagcatalog.ToolInvalidArgumentKey)
		}
		if _, exists := fields[key]; exists {
			return nil, faults.Errorf(diagcatalog.ToolDuplicateArgument, key)
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
		return nil, faults.Errorf(diagcatalog.ToolTrailingToolArguments)
	}
	return fields, nil
}

func ValidateArguments(definition Definition, body json.RawMessage) (map[string]json.RawMessage, error) {
	if len(body) > MaxArgumentsBytes || !utf8.Valid(body) {
		return nil, faults.Errorf(diagcatalog.ToolInvalidOrOversizedArguments)
	}
	fields, err := argumentObject(body)
	if err != nil {
		return nil, err
	}
	for name := range fields {
		if !slices.ContainsFunc(definition.Parameters, func(p Parameter) bool { return p.Name == name }) {
			return nil, faults.Errorf(diagcatalog.ToolUnexpectedArgument, definition.Name, name)
		}
	}
	for _, parameter := range definition.Parameters {
		value, present := fields[parameter.Name]
		if !present {
			if parameter.Required {
				return nil, faults.Errorf(diagcatalog.ToolMissingArgument, parameter.Name)
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
		return faults.Errorf(diagcatalog.ToolCannotBeNull, p.Name)
	}
	if p.Kind == Integer {
		var number int
		if json.Unmarshal(value, &number) != nil || number < 0 || number > p.Max {
			return faults.Errorf(diagcatalog.ToolRequiresAnIntegerFromTo, p.Name, p.Max)
		}
		return nil
	}
	if p.Kind == ObjectList {
		var entries []json.RawMessage
		if err := json.Unmarshal(value, &entries); err != nil || len(entries) == 0 || len(entries) > p.Max {
			return faults.Errorf(diagcatalog.ToolRequiresToObjects, p.Name, p.Max)
		}
		for i, entry := range entries {
			if _, err := ValidateArguments(Definition{Name: p.Name, Parameters: p.Fields}, entry); err != nil {
				return faults.Errorf(diagcatalog.ToolWrappedCause, p.Name, i, err)
			}
		}
		return nil
	}
	if p.Kind == TextList {
		var values []string
		if err := json.Unmarshal(value, &values); err != nil || !p.AllowEmpty && len(values) < 1 || len(values) > p.Max {
			return faults.Errorf(diagcatalog.ToolRequiresABoundedStringArray, p.Name)
		}
		for _, text := range values {
			if strings.TrimSpace(text) == "" || len(text) > p.MaxBytes || strings.ContainsRune(text, 0) || !utf8.ValidString(text) {
				return faults.Errorf(diagcatalog.ToolInvalidItem, p.Name)
			}
		}
		return nil
	}
	var text string
	if err := json.Unmarshal(value, &text); err != nil || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return faults.Errorf(diagcatalog.ToolRequiresAValidString, p.Name)
	}
	if len(text) > p.MaxBytes || !p.AllowEmpty && strings.TrimSpace(text) == "" {
		return faults.Errorf(diagcatalog.ToolRequiresTextWithinBytes, p.Name, p.MaxBytes)
	}
	if len(p.Values) > 0 && !slices.Contains(p.Values, text) {
		return faults.Errorf(diagcatalog.ToolMustBeOneOf, p.Name, strings.Join(p.Values, ", "))
	}
	switch p.Kind {
	case Text:
	case LiteralPath:
		if strings.ContainsAny(text, "\r\n") {
			return faults.Errorf(diagcatalog.ToolRequiresOneLiteralPathWithoutLineBreaks, p.Name)
		}
	default:
		return faults.Errorf(diagcatalog.ToolUnsupportedParameterKind)
	}
	return nil
}
