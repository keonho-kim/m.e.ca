// Purpose: Parse complete native arguments against the immutable concrete tool schemas.
// Pattern: Native tool input boundary.
// Usage: Executor preparation and parallel-wave classification call Parse.
// Related: internal/tools/local/catalog/catalog.go, internal/agent/tool/arguments.go.
package catalog

import (
	"encoding/json"
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local/textedit"
)

type Arguments struct {
	CommandID    string          "json:\"command_id\""
	WaitSeconds  *int            "json:\"wait_seconds,omitempty\""
	Paths        []string        `json:"paths"`
	FormatPolicy string          `json:"format_policy,omitempty"`
	Path         string          `json:"path"`
	Content      string          `json:"content"`
	Edits        []textedit.Edit `json:"edits"`
	Command      string          `json:"command"`
}

func Parse(call toolapi.Call) (Specification, Arguments, error) {
	if err := call.CheckIdentity(); err != nil {
		return Specification{}, Arguments{}, err
	}
	var spec Specification
	for _, candidate := range catalog {
		if candidate.Name == call.Name {
			spec = candidate
			break
		}
	}
	if spec.Name == "" {
		return spec, Arguments{}, fmt.Errorf("unknown tool: %s", call.Name)
	}
	if _, err := toolapi.ValidateArguments(toolapi.Definition{Name: spec.Name, Parameters: spec.fields}, call.Arguments); err != nil {
		return spec, Arguments{}, err
	}
	var input Arguments
	if err := json.Unmarshal(call.Arguments, &input); err != nil {
		return spec, input, err
	}
	return spec, input, nil
}
