// Purpose: Measure complete provider request encoding and streamed response handling.
// Pattern: Protocol workload benchmark.
// Usage: Run BenchmarkProviderRequests and BenchmarkProviderStream with -benchmem.
// Related: internal/provider/protocol/request.go, internal/provider/protocol/stream.go.
package protocol

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
)

func BenchmarkProviderRequests(b *testing.B) {
	request := modelapi.Request{ModelID: "served-model", MaxOutput: 8192}
	request.Tools = []toolapi.Definition{{Name: "execute_command", Description: "Execute a shell command.", Parameters: []toolapi.Parameter{{Name: "command", Kind: toolapi.Text, Required: true}}}}
	var messages []Message
	for range 40 {
		messages = append(messages, Message{Role: "user", Content: strings.Repeat("source code & 한국어 <module>\n", 128)})
	}
	for _, provider := range []modelconn.ProviderKind{modelconn.OpenRouter, modelconn.LMStudio} {
		b.Run(string(provider), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, body, err := Encode(provider, "https://provider.test/v1", nil, request, messages, Controls{})
				if err != nil || len(body) == 0 {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkProviderStream(b *testing.B) {
	body := strings.Repeat("data: {\"choices\":[{\"delta\":{\"content\":\"response 한국어\",\"reasoning\":\"check\"}}]}\n\n", 256) +
		"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":12000,\"completion_tokens\":256,\"cost\":0.123}}\n\n" +
		"data: [DONE]\n\n"
	b.ReportAllocs()
	b.SetBytes(int64(len(body)))
	for b.Loop() {
		response := &http.Response{Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body))}
		count := 0
		err := ReadResponse(context.Background(), response, modelconn.OpenRouter, func(event modelapi.Event) error {
			if event.Delta != "" {
				count++
			}
			return nil
		})
		if err != nil || count != 256 {
			b.Fatalf("events=%d: %v", count, err)
		}
	}
}
