// Purpose: Measure Claude continuation-prefix validation across a retained conversation.
// Pattern: Native request encoding benchmark with immutable fixtures.
// Usage: go test ./internal/provider/protocol -run '^$' -bench BenchmarkClaudeSignedHistory -benchmem.
// Related: internal/provider/protocol/claude_request.go, internal/provider/protocol/claude/continuation.go.
package protocol

import (
	"encoding/json"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol/claude"
)

func BenchmarkClaudeSignedHistory(b *testing.B) {
	text := strings.Repeat("source observations ", 64)
	state, err := json.Marshal(claude.Continuation{Version: 1, Prefix: strings.Repeat("0", 64), Blocks: []claude.Block{{Type: "thinking", Signature: "opaque"}, {Type: "text", Text: text}}})
	if err != nil {
		b.Fatal(err)
	}
	messages := []Message{{Role: "system", Content: "rules"}}
	for range 128 {
		messages = append(messages, Message{Role: "user", Content: "continue"}, Message{Role: "assistant", Content: text, continuation: state})
	}
	request := modelapi.Request{ModelID: "alias", MaxOutput: 32}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, _, err := Encode(connection.Claude, "https://example.com/v1", nil, request, messages, Controls{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCloudToolHistory(b *testing.B) {
	arguments := json.RawMessage(`{"content":"` + strings.Repeat("observed text ", 2048) + `"}`)
	var source []modelapi.Message
	for range 32 {
		source = append(source, modelapi.Message{Role: "assistant", Calls: []toolapi.Call{{ID: "call", Name: "write_file", Arguments: arguments}}}, modelapi.Message{Role: "tool", ToolCallID: "call", ToolName: "write_file", Content: "done"})
	}
	for _, kind := range []connection.ProviderKind{connection.Claude, connection.Gemini, connection.OpenAI} {
		b.Run(string(kind), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				messages, err := Messages(kind, source)
				if err != nil {
					b.Fatal(err)
				}
				if _, _, err := Encode(kind, "https://example.com", nil, modelapi.Request{ModelID: "alias", MaxOutput: 32}, messages, Controls{}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
