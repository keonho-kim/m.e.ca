// Purpose: Keep an immutable encoding of each unchanged conversation message.
// Pattern: Prefix reconciliation with immutable encodings and content fingerprints.
// Usage: Encoder updates the changed suffix and releases obsolete fragments.
// Related: internal/provider/protocol/encoder.go, internal/provider/protocol/lmstudio.go.
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

func (e *Encoder) update(messages []Message) error {
	unchanged := 0
	for unchanged < min(len(e.history), len(messages)) && e.history[unchanged].matches(messages[unchanged]) {
		unchanged++
	}
	for _, entry := range e.history[unchanged:] {
		e.bytes -= len(entry.json)
		e.retained -= entry.retainedBytes()
	}
	clear(e.history[unchanged:])
	e.history = e.history[:unchanged]
	for _, message := range messages[unchanged:] {
		content := message.encoded
		var err error
		if content != nil {
			// A matching source already owns its immutable provider encoding.
		} else if e.kind == connection.LMStudio {
			var input []map[string]any
			input, err = lmMessages([]Message{message})
			if err == nil && len(input) > 0 {
				content, err = wireJSON.Marshal(input)
			}
			if err == nil && len(input) > 0 {
				content = content[1 : len(content)-1]
			}
		} else {
			content, err = wireJSON.Marshal(message)
		}
		if err != nil {
			return err
		}
		entry := encodedMessage{json: content}
		if message.source != nil {
			entry.source, entry.hasSource = *message.source, true
		} else {
			entry.key = identifyWire(message)
		}
		e.history = append(e.history, entry)
		e.bytes += len(content)
		e.retained += entry.retainedBytes()
	}
	return nil
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
