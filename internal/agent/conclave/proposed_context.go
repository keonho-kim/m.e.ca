// Purpose: Expose a sibling's frozen proposal as unexecuted review context.
// Pattern: Pure review context projection.
// Usage: Review checks pending candidates before reading a not-yet-created disk file.
// Related: internal/agent/conclave/review.go, internal/agent/tool/candidate.go.
package conclave

import (
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"unicode/utf8"
)

func proposedContext(needed ContextRequest, pending []tool.FileDelta) (json.RawMessage, bool, error) {
	for _, file := range pending {
		if file.Path != needed.Path {
			continue
		}
		truncated := false
		if len(file.Diff) > needed.Limit {
			end := needed.Limit
			for end > 0 && !utf8.RuneStart(file.Diff[end]) {
				end--
			}
			file.Diff = file.Diff[:end]
			truncated = true
		}
		body, err := json.Marshal(struct {
			Executed  bool           `json:"executed"`
			Source    string         `json:"source"`
			Delta     tool.FileDelta `json:"proposed_delta"`
			Truncated bool           `json:"truncated"`
		}{false, "Frozen sibling proposal in this batch; not a disk read or execution result. Diff excerpt starts at the beginning, regardless of requested file offset.", file, truncated})
		return body, true, err
	}
	return nil, false, nil
}
