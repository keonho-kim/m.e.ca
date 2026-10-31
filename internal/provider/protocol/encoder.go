// Purpose: Reuse validated text-history encodings during one serial agent execution.
// Pattern: Execution-owned bounded prefix cache.
// Usage: A scoped HTTP generator owns Encoder and calls Clear after its final request.
// Related: internal/provider/protocol/encoded_history.go, internal/provider/protocol/request.go.
package protocol

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

const (
	cacheThreshold = 64 * 1024
	cacheCapacity  = 16 * 1024 * 1024
)

// Encoder is owned by one serial generation loop, never by a shared HTTP client.
// The bound limits additional retained encodings, not accepted context size.
type Encoder struct {
	kind     connection.ProviderKind
	model    string
	history  []encodedMessage
	bytes    int
	retained int
}

// Clear releases all references to encoded conversation history.
func (e *Encoder) Clear() { *e = Encoder{} }

// Encode preserves Encode's request values while reusing cached JSON fragments
// as replayable HTTP segments. Only fragments produced by wireJSON enter
// the cache; untrusted raw values still pass through its normal validation.
func (e *Encoder) Encode(kind connection.ProviderKind, baseURL string, info *connection.ModelInfo, request modelapi.Request, messages []Message, controls Controls) (string, Body, error) {
	if e == nil {
		return singleBody(kind, baseURL, info, request, messages, controls)
	}
	size, text := textHistorySize(messages)
	if !text || size < cacheThreshold || size > cacheCapacity {
		e.Clear()
		return singleBody(kind, baseURL, info, request, messages, controls)
	}
	if e.kind != kind || e.model != request.ModelID || len(messages) < len(e.history) {
		e.Clear()
	}
	e.kind, e.model = kind, request.ModelID
	if err := e.update(messages); err != nil {
		e.Clear()
		return "", Body{}, err
	}
	endpoint, field := baseURL+"/responses", `,"input":[`
	var header []byte
	var err error
	if kind == connection.LMStudio {
		header, err = wireJSON.Marshal(lmHeader(request.ModelID, request.MaxOutput, controls.ReasoningEffort, request.Temperature, request.Tools))
	} else {
		var payload wireRequest
		endpoint, payload = chatPayload(kind, baseURL, info, request, controls)
		header, err = wireJSON.Marshal(payload)
		field = `,"messages":[`
	}
	if err != nil {
		e.Clear()
		return "", Body{}, err
	}
	header = append(header[:len(header)-1], field...)
	parts := make([][]byte, 0, 2*len(e.history)+2)
	parts = append(parts, header)
	comma, end := []byte(","), []byte("]}")
	bodySize := len(header) + e.bytes + len(end)
	count := 0
	for _, entry := range e.history {
		if len(entry.json) == 0 {
			continue
		}
		if count > 0 {
			parts = append(parts, comma)
			bodySize++
		}
		parts = append(parts, entry.json)
		count++
	}
	if count == 0 {
		e.Clear()
		return singleBody(kind, baseURL, info, request, messages, controls)
	}
	if e.retained > cacheCapacity {
		// The returned body owns its fragments until transmission ends. Discard
		// cache keys now instead of rejecting or re-encoding a large request.
		e.Clear()
	}
	return endpoint, Body{parts: append(parts, end), size: bodySize}, nil
}

func singleBody(kind connection.ProviderKind, baseURL string, info *connection.ModelInfo, request modelapi.Request, messages []Message, controls Controls) (string, Body, error) {
	endpoint, content, err := Encode(kind, baseURL, info, request, messages, controls)
	return endpoint, Body{single: content, size: len(content)}, err
}

func textHistorySize(messages []Message) (int, bool) {
	total := 0
	for _, m := range messages {
		content, text := m.Content.(string)
		if !text || len(m.Images) > 0 {
			return 0, false
		}
		total += 512 + len(content) + len(m.Thinking) + len(m.Reasoning) + len(m.ReasoningContent) + len(m.Calls) + 32*len(m.ReasoningDetails)
		for _, raw := range m.ReasoningDetails {
			total += len(raw)
		}
	}
	return total, true
}
