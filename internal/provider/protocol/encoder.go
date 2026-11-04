// Purpose: Reuse validated conversation encodings during one serial agent execution.
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
	capacity int
	kind     connection.ProviderKind
	model    string
	history  []encodedMessage
	groups   []*sourceGroup
	retained int
	tools    toolSnapshot
}

// Clear releases all references to encoded conversation history.
func (e *Encoder) Clear() { *e = Encoder{capacity: e.capacity} }

// NewEncoder bounds retained wire encodings without restricting accepted input.
func NewEncoder(capacity int) Encoder { return Encoder{capacity: capacity} }
func (e *Encoder) limit() int {
	if e.capacity > 0 {
		return e.capacity
	}
	return cacheCapacity
}

func (e *Encoder) clearHistory() { e.history = nil; e.groups = nil; e.retained = 0 }

// Encode preserves Encode's request values while reusing cached JSON fragments
// as replayable HTTP segments. Only fragments produced by wireJSON enter
// the cache; untrusted raw values still pass through its normal validation.
func (e *Encoder) Encode(kind connection.ProviderKind, baseURL string, info *connection.ModelInfo, request modelapi.Request, messages []Message, controls Controls) (string, Body, error) {
	if e == nil {
		return singleBody(kind, baseURL, info, request, messages, controls)
	}
	if e.kind != kind || e.model != request.ModelID {
		e.Clear()
	}
	e.kind, e.model = kind, request.ModelID
	size, text := textHistorySize(messages)
	prepared := len(messages) > 0 && messages[0].source != nil
	if !prepared && (!text || size < cacheThreshold) {
		e.clearHistory()
		if len(request.Tools) == 0 {
			e.tools = toolSnapshot{}
			return singleBody(kind, baseURL, info, request, messages, controls)
		}
		return e.encodeWhole(kind, baseURL, info, request, messages, controls)
	}
	if !prepared {
		e.groups = nil
	}
	endpoint, header, err := e.header(kind, baseURL, info, request, controls)
	field := `,"input":[`
	if kind != connection.LMStudio {
		field = `,"messages":[`
	}
	if err != nil {
		e.Clear()
		return "", Body{}, err
	}
	parts, headerBytes := header.appendTo(make([][]byte, 0, 2*len(messages)+5), field)
	parts, historyBytes, err := e.appendHistory(parts, messages)
	if err != nil {
		e.Clear()
		return "", Body{}, err
	}
	end := []byte("]}")
	return endpoint, Body{parts: append(parts, end), size: headerBytes + historyBytes + len(end)}, nil
}

func singleBody(kind connection.ProviderKind, baseURL string, info *connection.ModelInfo, request modelapi.Request, messages []Message, controls Controls) (string, Body, error) {
	endpoint, content, err := Encode(kind, baseURL, info, request, messages, controls)
	return endpoint, Body{single: content, size: len(content)}, err
}

func textHistorySize(messages []Message) (int, bool) {
	total := 0
	for _, m := range messages {
		if m.source != nil && m.encoded != nil {
			total += m.sourceBytes
			continue
		}
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
