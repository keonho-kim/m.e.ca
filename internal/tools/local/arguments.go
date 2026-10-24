// Purpose: Validate runtime-built tool arguments against the native tool catalog.
// Pattern: Simple Module (execution trust boundary).
// Usage: Called by Executor.Prepare after complete native argument validation.
// Related: internal/tools/local/catalog.go, internal/agent/tool/arguments.go.
package local

import (
	"bytes"
	"encoding/json"
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local/textedit"
	"io"
)

type arguments struct {
	Paths        []string        `json:"paths"`
	FormatPolicy string          `json:"format_policy,omitempty"`
	Path         string          `json:"path"`
	Content      string          `json:"content"`
	Edits        []textedit.Edit `json:"edits"`
	Command      string          `json:"command"`
}

func parseArguments(call toolapi.Call) (specification, arguments, error) {
	if err := call.Check(); err != nil {
		return specification{}, arguments{}, err
	}
	var spec specification
	for _, candidate := range catalog {
		if candidate.name == call.Name {
			spec = candidate
			break
		}
	}
	if spec.name == "" {
		return spec, arguments{}, fmt.Errorf("unknown tool: %s", call.Name)
	}
	if _, err := toolapi.ValidateArguments(toolapi.Definition{Name: spec.name, Parameters: spec.fields}, call.Arguments); err != nil {
		return spec, arguments{}, err
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
