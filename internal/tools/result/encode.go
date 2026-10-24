// Purpose: Encode tool observations and summaries into the shared result contract.
// Pattern: Adapter result encoding.
// Usage: File, search and process adapters call Encode.
// Related: internal/agent/tool/execution.go.
package result

import (
	"encoding/json"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func Encode(value any, summary string) (toolapi.Result, error) {
	content, err := json.Marshal(value)
	return toolapi.Result{Content: content, Summary: summary}, err
}
