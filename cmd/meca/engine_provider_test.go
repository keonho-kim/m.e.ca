//go:build !minimum

// Purpose: Supply deterministic native streaming model/tool responses for the complete engine workload.
// Pattern: Loopback-only provider fixture with bounded request consumption.
// Usage: TestEngineWorkload and BenchmarkEngineWorkload share the same real HTTP client path.
// Related: cmd/meca/engine_workload_test.go, internal/provider/protocol/stream.go.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/keonho-kim/meca/internal/provider/connection"
)

type engineFrames struct {
	writer http.ResponseWriter
	err    error
}

func (f *engineFrames) send(kind string, value any) {
	if f.err != nil {
		return
	}
	body, err := json.Marshal(value)
	if err != nil {
		f.err = err
		return
	}
	if kind != "" {
		_, f.err = fmt.Fprintf(f.writer, "event: %s\n", kind)
	}
	if f.err == nil {
		_, f.err = fmt.Fprintf(f.writer, "data: %s\n\n", body)
	}
}

func engineProvider(w http.ResponseWriter, r *http.Request, kind connection.ProviderKind, step int) {
	defer r.Body.Close()
	if _, err := io.Copy(io.Discard, io.LimitReader(r.Body, 16<<20)); err != nil {
		http.Error(w, "fixture request failed", 400)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	frames := engineFrames{writer: w}
	arguments := map[string]string{"path": "fixture.txt"}
	call := map[string]any{"id": fmt.Sprintf("bench_call_%d", step), "name": "read_file", "args": arguments}
	text := "# Verified\n\n```go\n" + strings.Repeat("fmt.Println(\"observed\")\n", 1024) + "```\n\nCompleted."
	var chunks []string
	for offset := 0; offset < len(text); offset += 1024 {
		chunks = append(chunks, text[offset:min(len(text), offset+1024)])
	}
	tool := step < engineToolRounds
	switch kind {
	case connection.Claude:
		frames.send("message_start", map[string]any{"type": "message_start", "message": map[string]any{"usage": map[string]int{"input_tokens": 64, "output_tokens": 0}}})
		block := map[string]any{"type": "text", "text": ""}
		reason := "end_turn"
		if tool {
			block = map[string]any{"type": "tool_use", "id": call["id"], "name": "read_file", "input": map[string]any{}}
			reason = "tool_use"
		}
		frames.send("content_block_start", map[string]any{"type": "content_block_start", "index": 0, "content_block": block})
		if tool {
			encoded, err := json.Marshal(arguments)
			if err != nil {
				return
			}
			frames.send("content_block_delta", map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]string{"type": "input_json_delta", "partial_json": string(encoded)}})
		} else {
			for _, chunk := range chunks {
				frames.send("content_block_delta", map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]string{"type": "text_delta", "text": chunk}})
			}
		}
		frames.send("content_block_stop", map[string]any{"type": "content_block_stop", "index": 0})
		frames.send("message_delta", map[string]any{"type": "message_delta", "delta": map[string]string{"stop_reason": reason}, "usage": map[string]int{"output_tokens": 64}})
		frames.send("message_stop", map[string]string{"type": "message_stop"})
	case connection.Gemini:
		parts := []map[string]any{{"functionCall": call, "thoughtSignature": "benchmark-signature"}}
		if tool {
			frames.send("", engineGeminiFrame(parts, true))
		} else {
			for index, chunk := range chunks {
				frames.send("", engineGeminiFrame([]map[string]any{{"text": chunk}}, index == len(chunks)-1))
			}
		}
	default:
		if tool {
			encoded, err := json.Marshal(arguments)
			if err != nil {
				return
			}
			frames.send("", map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": call["id"], "type": "function", "function": map[string]any{"name": "read_file", "arguments": string(encoded)}}}}, "finish_reason": "tool_calls"}}})
		} else {
			for index, chunk := range chunks {
				var reason any
				if index == len(chunks)-1 {
					reason = "stop"
				}
				frames.send("", map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]string{"content": chunk}, "finish_reason": reason}}})
			}
		}
		if frames.err == nil {
			_, frames.err = io.WriteString(w, "data: [DONE]\n\n")
		}
	}
}

func engineGeminiFrame(parts []map[string]any, finished bool) map[string]any {
	candidate := map[string]any{"content": map[string]any{"role": "model", "parts": parts}, "index": 0}
	if finished {
		candidate["finishReason"] = "STOP"
	}
	return map[string]any{"candidates": []any{candidate}, "usageMetadata": map[string]int{"promptTokenCount": 64, "candidatesTokenCount": 64, "totalTokenCount": 128}}
}
