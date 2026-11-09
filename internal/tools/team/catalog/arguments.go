// Purpose: Parse team-only native arguments against the private immutable schemas.
// Pattern: Input boundary.
// Usage: The cooperative executor dispatches validated Arguments to membership and history operations.
// Related: internal/tools/team/catalog/catalog.go, internal/tools/team/coordination.go.
package catalog

import (
	"encoding/json"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Arguments struct {
	Message     string   `json:"message"`
	Member      string   `json:"member_name"`
	AfterMember string   `json:"after_member"`
	Summary     string   `json:"summary"`
	Contract    string   `json:"contract"`
	Mentions    []string `json:"mentions"`
	ReplyTo     int64    `json:"reply_to"`
	Paths       []string `json:"paths"`
	After       int64    `json:"after"`
}

func Parse(call toolapi.Call) (Arguments, error) {
	spec, ok := definition(call.Name)
	if !ok {
		return Arguments{}, faults.Errorf(diagcatalog.CatalogUnknownCooperativeTool)
	}
	if _, err := toolapi.ValidateArguments(spec, call.Arguments); err != nil {
		return Arguments{}, err
	}
	var input Arguments
	err := json.Unmarshal(call.Arguments, &input)
	return input, err
}
