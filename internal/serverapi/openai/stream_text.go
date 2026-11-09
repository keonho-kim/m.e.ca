// Purpose: Reuse one escaped final text across the four complete Responses terminal events.
// Pattern: Encoded JSON fragment composition at the wire boundary.
// Usage: encoder.finish supplies typed metadata with exactly one nil text leaf.
// Related: internal/serverapi/openai/responses.go, internal/serverapi/eventstream/writer.go.
package openai

import (
	"bytes"
	"encoding/json"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (e *encoder) sendText(kind string, fields map[string]any, quoted []byte) error {
	fields["type"], fields["sequence_number"] = kind, e.sequence
	e.sequence++
	body, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	// Metadata is encoded independently of public text. An escaped string
	// cannot impersonate this unescaped field marker. Fail closed if a future
	// DTO change removes or duplicates the single expected text placeholder.
	marker := []byte(`"text":null`)
	if bytes.Count(body, marker) != 1 {
		return faults.Errorf(diagcatalog.OpenaiTerminalEventMustContainOneTextPlaceholder)
	}
	start := bytes.Index(body, marker) + len(`"text":`)
	return e.writer.SendParts(kind, "", body[:start], quoted, body[start+len("null"):])
}
