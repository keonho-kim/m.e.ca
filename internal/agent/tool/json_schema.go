// Purpose: Derive ordered JSON tool schemas from the parameters used by runtime validation.
// Pattern: Pure projection.
// Usage: Provider adapters include Definition.JSONSchema in native function declarations.
// Related: internal/agent/tool/schema.go, internal/agent/tool/arguments.go.
package tool

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
)

func (d Definition) JSONSchema() map[string]any {
	required := make([]string, 0, len(d.Parameters))
	for _, p := range d.Parameters {
		if p.Required {
			required = append(required, p.Name)
		}
	}
	return map[string]any{"type": "object", "properties": schemaProperties(d.Parameters), "required": required, "additionalProperties": false}
}

// Schema order guides generation. Incoming arguments are objects in any order.
type schemaProperties []Parameter

func (properties schemaProperties) MarshalJSON() ([]byte, error) {
	var body bytes.Buffer
	encoder := jsontext.NewEncoder(&body)
	if err := encoder.WriteToken(jsontext.BeginObject); err != nil {
		return nil, err
	}
	for _, parameter := range properties {
		if err := encoder.WriteToken(jsontext.String(parameter.Name)); err != nil {
			return nil, err
		}
		if err := json.MarshalEncode(encoder, parameter.jsonSchema(), json.Deterministic(true)); err != nil {
			return nil, err
		}
	}
	if err := encoder.WriteToken(jsontext.EndObject); err != nil {
		return nil, err
	}
	return body.Bytes(), nil
}

func (p Parameter) jsonSchema() map[string]any {
	value := map[string]any{"type": "string", "description": p.Description}
	switch p.Kind {
	case Integer:
		value["type"], value["minimum"], value["maximum"] = "integer", 0, p.Max
	case ObjectList:
		value["type"], value["minItems"], value["maxItems"] = "array", 1, p.Max
		if p.AllowEmpty {
			value["minItems"] = 0
		}
		value["items"] = (Definition{Parameters: p.Fields}).JSONSchema()
	case TextList:
		value["type"], value["minItems"], value["maxItems"] = "array", 1, p.Max
		if p.AllowEmpty {
			value["minItems"] = 0
		}
		value["items"] = map[string]any{"type": "string", "minLength": 1, "maxLength": p.MaxBytes}
	default:
		maxLength := p.MaxBytes
		if p.Kind == SingleLineText {
			maxLength = p.Max
		}
		value["maxLength"] = maxLength
		if !p.AllowEmpty {
			value["minLength"] = 1
		}
	}
	if len(p.Values) > 0 {
		value["enum"] = p.Values
	}
	return value
}
