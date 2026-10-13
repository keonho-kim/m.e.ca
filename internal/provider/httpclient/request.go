// Purpose: Translate agent generation requests into native and compatible wire formats.
// Pattern: Adapter.
// Usage: Client.Generate and connection diagnostics.
// Related: internal/agent/model.go, internal/provider/httpclient/stream.go, internal/provider/httpclient/wire.go.

package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
)

func (client *Client) Generate(ctx context.Context, request agent.ModelRequest, emit func(agent.ModelEvent) error) (err error) {
	if emit == nil {
		return fmt.Errorf("a model event consumer is required")
	}
	if request.ModelID == "" || len(request.ModelID) > 512 || request.MaxOutput < 1 || request.MaxOutput > 65536 {
		return fmt.Errorf("invalid model request limits")
	}
	if len(request.Messages) == 0 || len(request.Messages) > 1000 {
		return fmt.Errorf("invalid message or tool count")
	}
	native := client.connection.Kind == modelconn.Ollama
	vllmGLM := client.connection.Kind == modelconn.VLLM && glmThinking(request.ModelID)
	if request.Thinking != nil && !native && !vllmGLM {
		return fmt.Errorf("Thinking ON/OFF control is unverified for this provider")
	}
	messages := make([]wireMessage, 0, len(request.Messages))
	for _, message := range request.Messages {
		if message.Role != "system" && message.Role != "user" && message.Role != "assistant" && message.Role != "tool" {
			return fmt.Errorf("invalid message role")
		}

		if len(message.Calls) > 0 || message.Role == "tool" {
			return fmt.Errorf("model context must project tool audit records as text observations")
		}
		messages = append(messages, wireMessage{Role: message.Role, Content: message.Content})
	}
	payload := wireRequest{Model: request.ModelID, Messages: messages, Stream: true}
	endpoint := client.url("chat/completions")
	if native {
		endpoint = client.url("chat")
		payload.Think = request.Thinking
		payload.Options = &ollamaOptions{NumPredict: request.MaxOutput}
	} else {
		payload.MaxTokens = request.MaxOutput
		if request.Thinking != nil && vllmGLM {
			payload.ChatTemplate = &chatTemplate{EnableThinking: *request.Thinking}
		}
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if len(content) > 4*1024*1024 {
		return fmt.Errorf("model context exceeds request size limit")
	}
	response, err := client.do(ctx, http.MethodPost, endpoint, bytes.NewReader(content))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, response.Body.Close()) }()
	return readResponse(ctx, response, native, emit)
}

func (client *Client) Diagnose(ctx context.Context, model string) (modelconn.Capabilities, error) {
	caps := modelconn.Capabilities{}
	var thinking *bool
	if supported, err := client.ThinkingControl(ctx, model); err == nil && supported {
		off := false
		thinking = &off
	}
	messages := []agent.Message{{Role: "system", Content: "You are testing a literal text tool dialogue. Do not use native tool calls or JSON."}, {Role: "user", Content: "Reply exactly TOOL read_file"}}
	for _, expected := range []string{"TOOL read_file", "AGENTS.md"} {
		var text strings.Builder
		var reasoning strings.Builder
		done := false
		err := client.Generate(ctx, agent.ModelRequest{ModelID: model, MaxOutput: 512, Messages: messages, Thinking: thinking}, func(e agent.ModelEvent) error {
			if text.Len()+len(e.Delta) > 1024 {
				return fmt.Errorf("diagnostic reply too large")
			}
			text.WriteString(e.Delta)
			reasoning.WriteString(e.Thinking)
			if e.Done {
				done = true
				caps.Streaming = e.Streaming
			}
			return nil
		})
		if err != nil {
			return caps, fmt.Errorf("connection diagnostic failed: %w", err)
		}
		caps.Chat = done
		if thinking != nil && !*thinking && reasoning.Len() > 0 {
			return caps, fmt.Errorf("server emitted reasoning with Thinking OFF; verify its chat template and reasoning parser")
		}
		if !done || strings.TrimSpace(text.String()) != expected {
			return caps, nil
		}
		messages = append(messages, agent.Message{Role: "assistant", Content: text.String()}, agent.Message{Role: "user", Content: "ARGUMENT read_file.path: Reply with only the literal path AGENTS.md, without quotes."})
		messages[0].Content = "You are testing a literal text tool dialogue.\n" + prompt.Argument("read_file", agent.ToolParameter{Name: "path", Kind: agent.LiteralPath, Required: true, MaxBytes: 4096})
	}
	caps.ToolDialogue = true
	return caps, nil
}

func sanitizeReason(reason string) error {
	if reason == "length" {
		return fmt.Errorf("Model response reached the output token limit")
	}
	if reason == "content_filter" || reason == "error" || reason == "abort" {
		return fmt.Errorf("Model response stopped: %s", strings.ReplaceAll(reason, "\n", ""))
	}
	return nil
}
