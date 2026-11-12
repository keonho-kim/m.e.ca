//go:build !tui

// Purpose: Provide deterministic free model responses and controllable streaming for browser tests.
// Pattern: Local HTTP model fixture.
// Usage: UI browser integration starts this server instead of contacting a real provider.
// Related: cmd/meca/ui_browser_test.go, cmd/meca/ui_tool_provider_test.go.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func uiProvider(t *testing.T, home string) *httptest.Server {
	sequence := newUIToolSequence()
	burst := &uiTextBurst{}
	release := make(chan struct{})
	var once sync.Once
	releaseThinking := make(chan struct{})
	var thinkingOnce sync.Once
	expandStream := make(chan struct{})
	var expandOnce sync.Once
	expandThinking, finishThinking := make(chan struct{}), make(chan struct{})
	var expandThinkingOnce, finishThinkingOnce sync.Once
	startTool, finishTool := make(chan struct{}), make(chan struct{})
	var startToolOnce, finishToolOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if burst.control(w, r) {
			return
		}
		if sequence.control(w, r) {
			return
		}
		if r.URL.Path == "/test/start-tool" {
			startToolOnce.Do(func() { close(startTool) })
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == "/test/finish-tool" {
			finishToolOnce.Do(func() { close(finishTool) })
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == "/test/release" {
			once.Do(func() { close(release) })
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == "/test/release-thinking" {
			thinkingOnce.Do(func() { close(releaseThinking) })
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == "/test/expand-stream" {
			expandOnce.Do(func() { close(expandStream) })
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == "/test/expand-thinking" {
			expandThinkingOnce.Do(func() { close(expandThinking) })
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == "/test/finish-thinking" {
			finishThinkingOnce.Do(func() { close(finishThinking) })
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/models") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":[{"id":"fixture-model","max_model_len":32768}]}`)
			return
		}
		if r.URL.Path == "/tokenize" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"count":1024,"max_model_len":32768}`)
			return
		}
		if r.Method != "POST" || !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			http.NotFound(w, r)
			return
		}
		var request struct {
			Stream   bool            `json:"stream"`
			Template map[string]bool `json:"chat_template_kwargs"`
			Messages []struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		text := "Browser fixture response"
		for i := len(request.Messages) - 1; i >= 0; i-- {
			if request.Messages[i].Role == "user" {
				var value string
				if json.Unmarshal(request.Messages[i].Content, &value) == nil {
					text = value
				}
				break
			}
		}
		hold := strings.Contains(text, "HOLD_FIXTURE")
		needsApproval := strings.Contains(text, "APPROVAL_FIXTURE")
		for _, message := range request.Messages {
			if message.Role == "tool" {
				needsApproval = false
			}
		}
		if len(text) > 256 {
			text = text[:256]
		}
		if !request.Stream {
			w.Header().Set("Content-Type", "application/json")
			message := map[string]string{"content": "Fixture response"}
			if request.Template["enable_thinking"] {
				message["reasoning"] = "Fixture reasoning"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": "stop"}}})
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if strings.Contains(text, "PARALLEL_FIXTURE") {
			completed := false
			for _, message := range request.Messages {
				completed = completed || message.Role == "tool"
			}
			uiParallelReply(w, completed)
			return
		}
		if strings.Contains(text, "REPEATED_TOOL_FIXTURE") {
			tools := 0
			for _, message := range request.Messages {
				if message.Role == "tool" {
					tools++
				}
			}
			sequence.reply(w, r, tools)
			return
		}
		if needsApproval {
			arguments, _ := json.Marshal(map[string]string{"path": filepath.Join(home, "approval-written.txt"), "content": "approved fixture\n"})
			body, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "fixture-write", "type": "function", "function": map[string]any{"name": "write_file", "arguments": string(arguments)}}}}, "finish_reason": "tool_calls"}}})
			fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", body)
			_ = http.NewResponseController(w).Flush()
			return
		}
		chunk := func(field, value string, finish any) {
			body, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{field: value}, "finish_reason": finish}}})
			fmt.Fprintf(w, "data: %s\n\n", body)
			_ = http.NewResponseController(w).Flush()
		}
		if burst.reply(w, r, text, chunk) {
			return
		}
		if strings.Contains(text, "TOOL_CYCLE_FIXTURE") {
			hasTool := false
			for _, message := range request.Messages {
				hasTool = hasTool || message.Role == "tool"
			}
			if !hasTool {
				chunk("reasoning", "Stable initial tool reasoning", nil)
				select {
				case <-startTool:
				case <-r.Context().Done():
					return
				}
				fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"fixture-read\",\"type\":\"function\",\"function\":{\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\\\"sample.go\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\ndata: [DONE]\n\n")
				_ = http.NewResponseController(w).Flush()
				return
			}
			chunk("reasoning", "Stable subsequent reasoning\n\n"+strings.Repeat("long_identifier_", 160), nil)
			chunk("content", "Intermediate response before final completion.\n\n```text\n"+strings.Repeat("unbroken_code_", 160)+"\n```", nil)
			select {
			case <-finishTool:
			case <-r.Context().Done():
				return
			}
			chunk("content", "\n\nFinal tool-cycle answer.", "stop")
			fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":32,\"completion_tokens\":64}}\n\ndata: [DONE]\n\n")
			return
		}
		if strings.Contains(text, "LONG_THINKING_FIXTURE") {
			chunk("reasoning", strings.Repeat("Initial reasoning paragraph\n\n", 64)+"First reasoning tail\n\n", nil)
			select {
			case <-expandThinking:
			case <-r.Context().Done():
				return
			}
			chunk("reasoning", strings.Repeat("Continued reasoning paragraph\n\n", 64)+"Second reasoning tail\n\n", nil)
			select {
			case <-finishThinking:
			case <-r.Context().Done():
				return
			}
			chunk("reasoning", strings.Repeat("Final reasoning paragraph\n\n", 64)+"Third reasoning tail", nil)
			<-r.Context().Done()
			return
		}
		if strings.Contains(text, "THINKING_FIXTURE") {
			chunk("reasoning", "## Plan\n\n**Inspect** the workspace", nil)
			select {
			case <-releaseThinking:
			case <-r.Context().Done():
				return
			}
			chunk("reasoning", " and verify the result.", nil)
		} else if request.Template["enable_thinking"] {
			chunk("reasoning", "Fixture reasoning", nil)
		}
		chunk("content", "## Fixture\n\n", nil)
		if strings.Contains(text, "LONG_STREAM_FIXTURE") {
			select {
			case <-expandStream:
			case <-r.Context().Done():
				return
			}
			chunk("content", strings.Repeat("Streaming paragraph\n\n", 100), nil)
			<-r.Context().Done()
			return
		}
		if hold {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
		waiting:
			for {
				select {
				case <-release:
					break waiting
				case <-r.Context().Done():
					return
				case <-ticker.C:
					chunk("content", " ·", nil)
				}
			}
		}
		chunk("content", "Completed: "+text, "stop")
		if strings.Contains(text, "METRICS_FIXTURE") {
			fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":13,\"completion_tokens\":7}}\n\n")
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		_ = http.NewResponseController(w).Flush()
	}))
	t.Cleanup(server.Close)
	return server
}
