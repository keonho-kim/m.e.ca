// Purpose: Locate stored message text and its immutable payload reference for bounded native history reads.
// Pattern: Selective text projection with caller-owned authenticated payload reads.
// Usage: TextPrefix reads only requested public text prefixes without hydrating private fields.
// Related: internal/storage/content/text_prefix.go, internal/storage/sqlite/text_prefix.go.
package content

import (
	"encoding/json"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func textSource(body, refsBody []byte, skipIncomplete bool) (json.RawMessage, string, error) {
	var message struct {
		Content    json.RawMessage `json:"content"`
		Incomplete bool            `json:"incomplete"`
	}
	if err := json.Unmarshal(body, &message); err != nil {
		return nil, "", err
	}
	if skipIncomplete && message.Incomplete {
		return nil, "", nil
	}
	var refs map[string]string
	if err := json.Unmarshal(refsBody, &refs); err != nil {
		return nil, "", err
	}
	reference := ""
	for key, id := range refs {
		var path []string
		if err := json.Unmarshal([]byte(key), &path); err != nil || len(path) == 0 {
			return nil, "", faults.Errorf(diagcatalog.ContentInvalidStoredPayloadPath)
		}
		if path[0] != "content" {
			continue
		}
		if len(path) != 1 || string(message.Content) != "null" || reference != "" || id == "" {
			return nil, "", faults.Errorf(diagcatalog.ContentInvalidStoredContentPlaceholder)
		}
		reference = id
	}
	return message.Content, reference, nil
}
