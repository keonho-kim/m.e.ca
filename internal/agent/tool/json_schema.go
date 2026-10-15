// Purpose: Derive JSON tool schemas from the same parameters used by runtime validation.
// Pattern: Pure projection.
// Usage: Provider adapters include Definition.JSONSchema in native function declarations.
// Related: internal/agent/tool/schema.go, internal/agent/tool/arguments.go.
package tool

func (d Definition) JSONSchema() map[string]any {
	properties := make(map[string]any)
	required := []string{}
	for _, p := range d.Parameters {
		value := map[string]any{"type": "string", "description": p.Description}
		if p.Kind == TextList {
			value["type"] = "array"
			value["minItems"] = 1
			value["maxItems"] = p.Max
			value["items"] = map[string]any{"type": "string", "minLength": 1, "maxLength": p.MaxBytes}
		} else if p.Kind == Integer {
			value["type"], value["minimum"], value["maximum"] = "integer", p.Min, p.Max
		} else {
			value["maxLength"] = p.MaxBytes
			if p.MaxChars > 0 {
				value["maxLength"] = p.MaxChars
			}
			if !p.AllowEmpty {
				value["minLength"] = 1
			}
		}
		if p.Kind == LineAnchor {
			value["pattern"] = "^[1-9][0-9]*#[0-9a-f]{8}$"
		}
		if len(p.Values) > 0 {
			value["enum"] = p.Values
		}
		properties[p.Name] = value
		if p.Required {
			required = append(required, p.Name)
		}
	}
	return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
}
