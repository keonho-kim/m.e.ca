// Purpose: Parse team-only native arguments against the private immutable schemas.
// Pattern: Input boundary.
// Usage: The cooperative executor dispatches validated Arguments to membership and history operations.
// Related: internal/tools/team/catalog/catalog.go, internal/tools/team/coordination.go.
package catalog

import (
	"encoding/json"
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type Arguments struct {
	Message     string   `json:"message"`
	Member      string   `json:"member_name"`
	AfterMember string   `json:"after_member"`
	Summary     string   `json:"summary"`
	Paths       []string `json:"paths"`
	After       int64    `json:"after"`
}

func Parse(call toolapi.Call) (Arguments, error) {
	spec, ok := definition(call.Name)
	if !ok {
		return Arguments{}, fmt.Errorf("unknown cooperative tool")
	}
	if _, err := toolapi.ValidateArguments(spec, call.Arguments); err != nil {
		return Arguments{}, err
	}
	var input Arguments
	err := json.Unmarshal(call.Arguments, &input)
	return input, err
}
