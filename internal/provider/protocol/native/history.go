// Purpose: Decode the validated native function-call history shared by concrete request adapters.
// Pattern: Strict conversation-to-wire projection.
// Usage: Claude, Gemini and Responses builders reconstruct calls using engine-owned identities.
// Related: internal/provider/protocol/tools.go, internal/agent/tool/call.go.
package native

import (
	"encoding/json"
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func History(raw json.RawMessage) ([]toolapi.Call, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var values []struct {
		ID       string
		Function struct{ Name, Arguments string }
	}
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("invalid function call history")
	}
	if len(values) > toolapi.MaxCalls {
		return nil, fmt.Errorf("too many function calls")
	}
	calls := make([]toolapi.Call, len(values))
	for i, value := range values {
		calls[i] = toolapi.Call{ID: value.ID, Name: value.Function.Name, Arguments: json.RawMessage(value.Function.Arguments)}
		if err := calls[i].Check(); err != nil {
			return nil, err
		}
	}
	return calls, nil
}
