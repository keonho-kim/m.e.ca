// Purpose: Parse complete native arguments against the immutable concrete tool schemas.
// Pattern: Native tool input boundary.
// Usage: Executor preparation and parallel-wave classification call Parse.
// Related: internal/tools/local/catalog/catalog.go, internal/agent/tool/arguments.go.
package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local/textedit"
	"io"
)

type Arguments struct {
	Paths        []string        `json:"paths"`
	FormatPolicy string          `json:"format_policy,omitempty"`
	Path         string          `json:"path"`
	Content      string          `json:"content"`
	Edits        []textedit.Edit `json:"edits"`
	Command      string          `json:"command"`
}

func Parse(call toolapi.Call) (Specification, Arguments, error) {
	if err := call.Check(); err != nil {
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
