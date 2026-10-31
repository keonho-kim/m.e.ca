// Purpose: Keep an immutable encoding of each unchanged text-history message.
// Pattern: Prefix reconciliation with owned mutable-byte snapshots.
// Usage: Encoder updates the changed suffix and releases obsolete fragments.
// Related: internal/provider/protocol/encoder.go, internal/provider/protocol/lmstudio.go.
package protocol

import (
	"bytes"
	"encoding/json"

	"github.com/keonho-kim/meca/internal/provider/connection"
)

type encodedMessage struct {
	message Message
	json    []byte
}

func (e *Encoder) update(messages []Message) error {
	unchanged := 0
	for unchanged < min(len(e.history), len(messages)) && sameTextMessage(e.history[unchanged].message, messages[unchanged]) {
		unchanged++
	}
	for _, entry := range e.history[unchanged:] {
		e.bytes -= len(entry.json)
		e.retained -= entry.retainedBytes()
	}
	clear(e.history[unchanged:])
	e.history = e.history[:unchanged]
	for _, message := range messages[unchanged:] {
		var content []byte
		var err error
		if e.kind == connection.LMStudio {
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
		// RawMessage slices are caller-owned and can be overwritten on the next
		// request. Snapshot them so an in-place mutation invalidates the cache.
		message.Calls = bytes.Clone(message.Calls)
		if message.ReasoningDetails != nil {
			details := make([]json.RawMessage, len(message.ReasoningDetails))
			for i, raw := range message.ReasoningDetails {
				details[i] = bytes.Clone(raw)
			}
			message.ReasoningDetails = details
		}
		entry := encodedMessage{message: message, json: content}
		e.history = append(e.history, entry)
		e.bytes += len(content)
		e.retained += entry.retainedBytes()
	}
	return nil
}

func (e encodedMessage) retainedBytes() int {
	// Include conservative per-entry overhead and the mutable-byte snapshots.
	total := 512 + len(e.json) + len(e.message.Calls) + 32*len(e.message.ReasoningDetails)
	for _, raw := range e.message.ReasoningDetails {
		total += len(raw)
	}
	return total
}

func sameTextMessage(a, b Message) bool {
	if a.Role != b.Role || a.Content != b.Content || a.ToolCallID != b.ToolCallID || a.ToolName != b.ToolName || a.Thinking != b.Thinking || a.Reasoning != b.Reasoning || a.ReasoningContent != b.ReasoningContent || !bytes.Equal(a.Calls, b.Calls) || len(a.ReasoningDetails) != len(b.ReasoningDetails) {
		return false
	}
	for i, raw := range a.ReasoningDetails {
		if !bytes.Equal(raw, b.ReasoningDetails[i]) {
			return false
		}
	}
	return true
}
