// Purpose: Reuse one bounded tool-schema encoding without trusting mutable caller slices.
// Pattern: Execution-owned immutable snapshot with structural invalidation.
// Usage: Generation headers and exact vLLM tokenization share Encoder.EncodedTools.
// Related: internal/provider/protocol/encoder.go, internal/provider/protocol/header.go.
package protocol

import (
	"encoding/json"
	"reflect"
	"slices"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

const toolCacheCapacity = 256 * 1024

type toolSnapshot struct {
	kind        connection.ProviderKind
	definitions []toolapi.Definition
	encoded     json.RawMessage
	bytes       int
}

// EncodedTools returns immutable wire JSON; nil encoders perform no retention.
func (e *Encoder) EncodedTools(kind connection.ProviderKind, definitions []toolapi.Definition) (json.RawMessage, error) {
	if len(definitions) == 0 {
		if e != nil {
			e.tools = toolSnapshot{}
		}
		return nil, nil
	}
	if e != nil && e.tools.kind == kind && reflect.DeepEqual(e.tools.definitions, definitions) {
		return e.tools.encoded, nil
	}
	if e != nil {
		e.tools = toolSnapshot{}
	}
	var value any
	if kind == connection.LMStudio {
		value = lmHeader("", 0, "", nil, definitions).Tools
	} else {
		value = Tools(definitions)
	}
	encoded, err := wireJSON.Marshal(value)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return encoded, nil
	}
	retained := len(encoded)
	for _, definition := range definitions {
		retained += 128 + len(definition.Name) + len(definition.Description) + parameterBytes(definition.Parameters)
	}
	if e != nil && retained <= min(toolCacheCapacity, e.limit()) {
		owned := slices.Clone(definitions)
		for i := range owned {
			owned[i].Parameters = copyParameters(owned[i].Parameters)
		}
		e.tools = toolSnapshot{kind: kind, definitions: owned, encoded: encoded, bytes: retained}
	}
	return encoded, nil
}

func copyParameters(source []toolapi.Parameter) []toolapi.Parameter {
	out := slices.Clone(source)
	for i := range out {
		out[i].Fields = copyParameters(out[i].Fields)
		out[i].Values = slices.Clone(out[i].Values)
	}
	return out
}

func parameterBytes(parameters []toolapi.Parameter) int {
	total := 0
	for _, p := range parameters {
		total += 256 + len(p.Name) + len(p.Description) + len(p.Kind) + parameterBytes(p.Fields)
		for _, v := range p.Values {
			total += 16 + len(v)
		}
	}
	return total
}
