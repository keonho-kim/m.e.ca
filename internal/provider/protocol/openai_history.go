// Purpose: Replay completed OpenAI output and encrypted reasoning alongside current engine-owned calls.
// Pattern: Stateless Responses history projection.
// Usage: responseInput reconstructs only continuation bound to the selected provider and model.
// Related: internal/provider/protocol/responses/continuation.go, internal/provider/httpclient/request.go.
package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
	"strings"
)

func openAIHistory(message Message) ([]map[string]any, error) {
	var items []map[string]any
	decoder := json.NewDecoder(bytes.NewReader(message.continuation))
	decoder.UseNumber()
	if err := decoder.Decode(&items); err != nil || items == nil || !json.Valid(message.continuation) {
		return nil, fmt.Errorf("invalid OpenAI continuation")
	}
	calls, err := message.functionCalls()
	if err != nil {
		return nil, err
	}
	at := 0
	var text strings.Builder
	for _, item := range items {
		switch item["type"] {
		case "reasoning":
		case "function_call":
			if at >= len(calls) || item["call_id"] != calls[at].ID || item["name"] != calls[at].Name {
				return nil, fmt.Errorf("OpenAI continuation differs from tool history")
			}
			arguments, ok := item["arguments"].(string)
			if !ok || !native.SameArguments(arguments, string(calls[at].Arguments)) {
				return nil, fmt.Errorf("OpenAI continuation arguments differ from tool history")
			}
			at++
		case "message":
			parts, ok := item["content"].([]any)
			if !ok {
				return nil, fmt.Errorf("invalid OpenAI message content")
			}
			for _, value := range parts {
				part, ok := value.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("invalid OpenAI content part")
				}
				field := "text"
				if part["type"] == "refusal" {
					field = "refusal"
				}
				value, ok := part[field].(string)
				if !ok {
					return nil, fmt.Errorf("unsupported OpenAI content part")
				}
				text.WriteString(value)
			}
		default:
			return nil, fmt.Errorf("unsupported OpenAI continuation item")
		}
	}
	if at != len(calls) {
		return nil, fmt.Errorf("OpenAI continuation omitted a tool call")
	}
	content, ok := message.Content.(string)
	if !ok || text.String() != content && strings.TrimSpace(text.String()) != content {
		return nil, fmt.Errorf("OpenAI continuation text differs from message history")
	}
	// Keep item and text boundaries, argument serialization and opaque metadata intact.
	return items, nil
}
