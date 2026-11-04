// Purpose: Reuse a bounded immutable prefix while encoding every uncached request fragment.
// Pattern: Prefix reconciliation with request-owned overflow segments.
// Usage: Encoder keeps only complete source groups that fit alongside its tool schema.
// Related: internal/provider/protocol/encoder.go, internal/provider/protocol/response_input.go.
package protocol

import (
	"github.com/keonho-kim/meca/internal/provider/connection"
)

type encodedMessage struct {
	key       wireIdentity
	source    sourceIdentity
	hasSource bool
	json      []byte
}

func (e *Encoder) appendHistory(parts [][]byte, messages []Message) ([][]byte, int, error) {
	previous := e.history
	unchanged := 0
	for unchanged < min(len(previous), len(messages)) && previous[unchanged].matches(messages[unchanged]) {
		unchanged++
	}
	if unchanged == len(messages) && unchanged == len(previous) && e.retained+e.tools.bytes <= e.limit() {
		// The normal fully cached request needs no cache reconciliation.
		bytes, count := 0, 0
		comma := []byte(",")
		for _, entry := range previous {
			if len(entry.json) == 0 {
				continue
			}
			if count > 0 {
				parts = append(parts, comma)
				bytes++
			}
			parts = append(parts, entry.json)
			bytes += len(entry.json)
			count++
		}
		return parts, bytes, nil
	}
	e.history, e.retained = e.history[:0], 0
	bytes, count := 0, 0
	comma := []byte(",")
	for i, message := range messages {
		var entry encodedMessage
		if i < unchanged {
			entry = previous[i]
		} else {
			content, err := encodeMessage(e.kind, message)
			if err != nil {
				return nil, 0, err
			}
			entry.json = content
			if message.source != nil {
				entry.source, entry.hasSource = *message.source, true
			} else {
				entry.key = identifyWire(message)
			}
		}
		if len(entry.json) > 0 {
			if count > 0 {
				parts = append(parts, comma)
				bytes++
			}
			parts = append(parts, entry.json)
			bytes += len(entry.json)
			count++
		}
		// Once the prefix fills, overflow belongs only to the replayable Body.
		if i == len(e.history) && e.retained+entry.retainedBytes()+e.tools.bytes <= e.limit() {
			e.history = append(e.history, entry)
			e.retained += entry.retainedBytes()
		}
	}
	if len(previous) > len(e.history) {
		clear(previous[len(e.history):])
	}
	e.trimGroups()
	if len(e.history) == 0 {
		e.history = nil
	}
	return parts, bytes, nil
}

func encodeMessage(kind connection.ProviderKind, message Message) ([]byte, error) {
	if message.encoded != nil {
		return message.encoded, nil
	}
	if kind != connection.LMStudio {
		return wireJSON.Marshal(message)
	}
	input, err := responseInput([]Message{message})
	if err != nil || len(input) == 0 {
		return nil, err
	}
	content, err := wireJSON.Marshal(input)
	if err != nil {
		return nil, err
	}
	return content[1 : len(content)-1], nil
}

// Never retain keys for a partly cached multimodal/native group. Those keys
// can otherwise hold an oversized image or tool result through one small DTO.
func (e *Encoder) trimGroups() {
	keep := 0
	for keep < len(e.groups) && e.groups[keep].from+e.groups[keep].count <= len(e.history) {
		keep++
	}
	if keep < len(e.groups) && e.groups[keep].from < len(e.history) {
		from := e.groups[keep].from
		for _, entry := range e.history[from:] {
			e.retained -= entry.retainedBytes()
		}
		clear(e.history[from:])
		e.history = e.history[:from]
	}
	clear(e.groups[keep:])
	e.groups = e.groups[:keep]
	if keep == 0 {
		e.groups = nil
	} else if cap(e.groups) > max(16, keep*2) {
		e.groups = append([]*sourceGroup(nil), e.groups...)
	}
}

func (e encodedMessage) retainedBytes() int {
	return 1024 + len(e.json)
}

func (e encodedMessage) matches(message Message) bool {
	if message.source != nil {
		return e.hasSource && e.source == *message.source
	}
	return !e.hasSource && e.key == identifyWire(message)
}
