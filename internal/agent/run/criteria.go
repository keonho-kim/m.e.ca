// Purpose: Separate model-proposed completion criteria from executable tool arguments.
// Pattern: Native metadata boundary.
// Usage: Context advertises withCriteria; action captures validated criteria in its checkpoint.
// Related: internal/agent/run/checkpoint.go, internal/agent/run/context.go, internal/agent/run/action.go.
package run

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

const maxCriteria = 3
const maxCriterionBytes = 512

func withCriteria(definition toolapi.Definition) toolapi.Definition {
	definition.Parameters = append(append([]toolapi.Parameter(nil), definition.Parameters...), toolapi.Parameter{
		Name: "completion_criteria", Description: "Optional 1-3 requested outcomes, saved in context as model claims. Omit when unchanged.",
		Kind: toolapi.TextList, Max: maxCriteria, MaxBytes: maxCriterionBytes,
	})
	return definition
}

// ExtractIntent has already rejected duplicate keys and removed the reason field.
func extractCriteria(call toolapi.Call) (toolapi.Call, []string, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(call.Arguments, &fields); err != nil {
		return call, nil, err
	}
	raw, exists := fields["completion_criteria"]
	if !exists {
		return call, nil, nil
	}
	var criteria []string
	if err := json.Unmarshal(raw, &criteria); err != nil {
		return call, nil, fmt.Errorf("completion_criteria must be a short string array")
	}
	if len(criteria) == 0 {
		return call, nil, fmt.Errorf("completion_criteria requires 1-3 outcomes")
	}
	if err := checkCriteria(criteria); err != nil {
		return call, nil, err
	}
	delete(fields, "completion_criteria")
	encoded, err := json.Marshal(fields)
	if err != nil {
		return call, nil, err
	}
	call.Arguments = encoded
	return call, criteria, nil
}

func checkCriteria(criteria []string) error {
	if len(criteria) > maxCriteria {
		return fmt.Errorf("completion_criteria exceeds 3 outcomes")
	}
	seen := map[string]bool{}
	for _, criterion := range criteria {
		text := strings.TrimSpace(criterion)
		if text == "" || len(criterion) > maxCriterionBytes || !utf8.ValidString(criterion) || strings.ContainsRune(criterion, 0) || seen[text] {
			return fmt.Errorf("completion_criteria requires distinct nonempty UTF-8 outcomes within 512 bytes each")
		}
		seen[text] = true
	}
	return nil
}
