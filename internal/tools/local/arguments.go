// Purpose: Validate runtime-built tool arguments against the same ordered catalog used by dialogue.
// Pattern: Simple Module (execution trust boundary).
// Usage: Called by Executor.Prepare after staged argument collection.
// Related: internal/tools/local/catalog.go, internal/agent/dialogue/dialogue.go.
package local

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/dialogue"
	"io"
)

type arguments struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	OldText     string `json:"old_text"`
	NewText     string `json:"new_text"`
	Destination string `json:"destination"`
	Pattern     string `json:"pattern"`
	Command     string `json:"command"`
	Offset      int    `json:"offset"`
	Limit       int    `json:"limit"`
	Timeout     int    `json:"timeout_seconds"`
}

func parseArguments(call agent.ToolCall) (specification, arguments, error) {
	if err := call.Check(); err != nil {
		return specification{}, arguments{}, err
	}
	var spec specification
	for _, candidate := range specifications() {
		if candidate.name == call.Name {
			spec = candidate
			break
		}
	}
	if spec.name == "" {
		return spec, arguments{}, fmt.Errorf("unknown tool: %s", call.Name)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(call.Arguments, &fields); err != nil {
		return spec, arguments{}, err
	}
	for name := range fields {
		found := false
		for _, p := range spec.fields {
			if p.Name == name {
				found = true
				break
			}
		}
		if !found {
			return spec, arguments{}, fmt.Errorf("unexpected %s argument: %s", call.Name, name)
		}
	}
	for _, p := range spec.fields {
		value, ok := fields[p.Name]
		if !ok {
			if p.Required {
				return spec, arguments{}, fmt.Errorf("missing %s argument", p.Name)
			}
			continue
		}
		if err := dialogue.ValidateArgument(p, value); err != nil {
			return spec, arguments{}, err
		}
	}
	var input arguments
	decoder := json.NewDecoder(bytes.NewReader(call.Arguments))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return spec, input, err
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return spec, input, fmt.Errorf("trailing tool arguments")
	}
	return spec, input, nil
}
