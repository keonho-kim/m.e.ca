// Purpose: Restore only committed assistant content without reserializing a complete JSON record.
// Pattern: Selective text projection with caller-owned authenticated payload reads.
// Usage: SQLite ResponseOutput reads original public text and skips incomplete drafts before hydration.
// Related: internal/storage/sqlite/response_output.go, internal/storage/content/payload.go.
package content

import (
	"encoding/json"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func Text(body, refsBody []byte, load func(string) (json.RawMessage, error)) (string, error) {
	raw, reference, err := textSource(body, refsBody, true)
	if err == nil && reference != "" {
		id, quoted := Reference(reference)
		raw, err = load(id)
		if quoted {
			return string(raw), err
		}
	}
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return "", err
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return "", err
	}
	return text, nil
}

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
