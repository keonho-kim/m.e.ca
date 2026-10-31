// Purpose: Estimate text context bytes without serializing history or copying image bodies.
// Pattern: Pure fallback measurement.
// Usage: Compaction accounting uses TextBytes only when transport bytes and token usage are absent.
// Related: internal/agent/run/request_usage.go, internal/provider/httpclient/request.go.
package model

import "encoding/json"

// TextBytes counts text, continuation base64 size and argument bytes with small
// framing allowances. This is a compaction estimate, never displayed token usage.
func TextBytes(request Request) int64 {
	n := int64(32)
	for _, m := range request.Messages {
		n += int64(128 + len(m.Role) + len(m.Content) + len(m.Reasoning) + len(m.ID) + len(m.ModelID) + len(m.ProviderID) + len(m.ToolCallID) + len(m.ToolName) + len(m.ReasoningEffort))
		n += int64((len(m.ReasoningState) + 2) / 3 * 4)
		for _, call := range m.Calls {
			n += int64(64 + len(call.ID) + len(call.Name) + len(call.Arguments))
		}
	}
	// Tool declarations are small and independent of conversation length.
	if body, err := json.Marshal(request.Tools); err == nil {
		n += int64(len(body))
	}
	return n
}
