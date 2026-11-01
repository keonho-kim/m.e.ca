// Purpose: Project one refined plan into advisory implementation context without asserting completed work.
// Pattern: Pure context projection.
// Usage: Runtime and restored session history derive the same message from a lat_plan event.
// Related: internal/agent/run/lat.go, internal/storage/sqlite/context.go.
package lat

import (
	"encoding/json"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func Plan(text string) modelapi.Message {
	quoted, _ := json.Marshal(text)
	return modelapi.Message{Role: "system", Content: "Refined implementation plan for the current user request (advisory proposed steps, not completed work or new requirements):\n" + string(quoted) + "\nCurrent and later user directions take precedence over this plan. Use only steps supported by the user's request and actual evidence, adapt or discard superseded steps, and preserve scope and prior user decisions. Verify implementation and results before reporting completion."}
}
