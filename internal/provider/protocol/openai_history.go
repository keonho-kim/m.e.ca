// Purpose: Replay completed OpenAI output and encrypted reasoning alongside current engine-owned calls.
// Pattern: Stateless Responses history projection.
// Usage: responseInput reconstructs only continuation bound to the selected provider and model.
// Related: internal/provider/protocol/responses/continuation.go, internal/provider/httpclient/request.go.
package protocol

import (
	"bytes"
	"encoding/json"

	"strings"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
)

func openAIHistory(message Message) ([]map[string]any, error) {
	var items []map[string]any
	decoder := json.NewDecoder(bytes.NewReader(message.continuation))
	decoder.UseNumber()
	if err := decoder.Decode(&items); err != nil || items == nil || !json.Valid(message.continuation) {
		return nil, faults.Errorf(diagcatalog.ProtocolInvalidOpenAIContinuation)
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
				return nil, faults.Errorf(diagcatalog.ProtocolOpenAIContinuationDiffersFromToolHistory)
			}
			arguments, ok := item["arguments"].(string)
			if !ok || !native.SameArguments(arguments, string(calls[at].Arguments)) {
				return nil, faults.Errorf(diagcatalog.ProtocolOpenAIContinuationArgumentsDifferFromToolHistory)
			}
			at++
		case "message":
			parts, ok := item["content"].([]any)
			if !ok {
				return nil, faults.Errorf(diagcatalog.ProtocolInvalidOpenAIMessageContent)
			}
			for _, value := range parts {
				part, ok := value.(map[string]any)
				if !ok {
					return nil, faults.Errorf(diagcatalog.ProtocolInvalidOpenAIContentPart)
				}
				field := "text"
				if part["type"] == "refusal" {
					field = "refusal"
				}
				value, ok := part[field].(string)
				if !ok {
					return nil, faults.Errorf(diagcatalog.ProtocolUnsupportedOpenAIContentPart)
				}
				text.WriteString(value)
			}
		default:
			return nil, faults.Errorf(diagcatalog.ProtocolUnsupportedOpenAIContinuationItem)
		}
	}
	if at != len(calls) {
		return nil, faults.Errorf(diagcatalog.ProtocolOpenAIContinuationOmittedAToolCall)
	}
	content, ok := message.Content.(string)
	if !ok || text.String() != content && strings.TrimSpace(text.String()) != content {
		return nil, faults.Errorf(diagcatalog.ProtocolOpenAIContinuationTextDiffersFromMessageHistory)
	}
	// Keep item and text boundaries, argument serialization and opaque metadata intact.
	return items, nil
}
