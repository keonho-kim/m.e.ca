// Purpose: Assemble already-encoded chronological records without parsing their JSON again.
// Pattern: Pure ordered reduction with one owned output buffer.
// Usage: Independent memory plans and review tails encode original records before calling sourcePacket.
// Related: internal/agent/lat/memory/packet.go, internal/agent/lat/memory/tail.go.
package memory

import "encoding/json"

func sourcePacket(memoryKey, recordsKey, previous string, records []json.RawMessage) ([]byte, error) {
	quoted, err := json.Marshal(previous)
	if err != nil {
		return nil, err
	}
	// Both keys are private constant field names; records come directly from
	// json.Marshal in records(), never from unvalidated persisted JSON.
	size := len(memoryKey) + len(recordsKey) + len(quoted) + 16
	for _, record := range records {
		size += len(record) + 1
	}
	body := make([]byte, 0, size)
	body = append(body, `{"`...)
	body = append(body, memoryKey...)
	body = append(body, `":`...)
	body = append(body, quoted...)
	body = append(body, `,"`...)
	body = append(body, recordsKey...)
	body = append(body, `":`...)
	if records == nil {
		body = append(body, "null"...)
	} else {
		body = append(body, '[')
		for i, record := range records {
			if i > 0 {
				body = append(body, ',')
			}
			body = append(body, record...)
		}
		body = append(body, ']')
	}
	return append(body, '}'), nil
}
