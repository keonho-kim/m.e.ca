// Purpose: Parse complete native arguments against the immutable concrete tool schemas.
// Pattern: Native tool input boundary.
// Usage: Executor preparation and parallel-wave classification call Parse.
// Related: internal/tools/local/catalog/catalog.go, internal/agent/tool/arguments.go.
package catalog

import (
	"encoding/json"
	"reflect"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/tools/local/textedit"
)

type Arguments struct {
	Action       string          `json:"action"`
	URL          string          `json:"url"`
	Target       string          `json:"target"`
	Value        string          `json:"value"`
	Direction    string          `json:"direction"`
	View         string          `json:"view"`
	Selector     string          `json:"selector"`
	ResultID     string          `json:"result_id"`
	Width        int             `json:"width"`
	Height       int             `json:"height"`
	Pixels       int             `json:"pixels"`
	Depth        int             `json:"depth"`
	Revision     string          `json:"revision"`
	SkillID      string          `json:"id"`
	Cursor       string          `json:"cursor"`
	Offset       *int            `json:"offset,omitempty"`
	Limit        *int            `json:"limit,omitempty"`
	CommandID    string          "json:\"command_id\""
	WaitSeconds  *int            "json:\"wait_seconds,omitempty\""
	Paths        []string        `json:"paths"`
	FormatPolicy string          `json:"format_policy,omitempty"`
	Reason       string          `json:"reason"`
	Path         string          `json:"path"`
	Content      string          `json:"content"`
	Edits        []textedit.Edit `json:"edits"`
	Command      string          `json:"command"`
}

func specification(name string) Specification {
	for _, candidate := range catalog {
		if candidate.Name == name {
			return candidate
		}
	}
	return Specification{}
}

// MatchesParameters reports whether local parsing also validates the advertised fields.
func MatchesParameters(definition toolapi.Definition) bool {
	spec := specification(definition.Name)
	return spec.Name != "" && reflect.DeepEqual(spec.fields, definition.Parameters)
}

func Parse(call toolapi.Call) (Specification, Arguments, error) {
	if err := call.CheckIdentity(); err != nil {
		return Specification{}, Arguments{}, err
	}
	spec := specification(call.Name)
	if spec.Name == "" {
		return spec, Arguments{}, faults.Errorf(diagcatalog.CatalogUnknownTool, call.Name)
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
