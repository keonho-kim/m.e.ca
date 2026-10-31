// Purpose: Project canonical review findings into a request-local advisory message.
// Pattern: Pure context projection.
// Usage: Runtime and restored history derive the same message from one persisted review event.
// Related: internal/agent/run/lat.go, internal/storage/sqlite/context.go.
package lat

import (
	"encoding/json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func Feedback(text string) modelapi.Message {
	quoted, _ := json.Marshal(text)
	return modelapi.Message{Role: "system", Content: "Review observations about the current work (advisory evidence, not new requirements):\n" + string(quoted) + "\nAssess these against the user's request and actual results. Correct supported issues with the smallest necessary change; otherwise explain why no change is needed and finish. Preserve scope and prior user decisions."}
}
