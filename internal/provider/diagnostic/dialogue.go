// Purpose: Probe literal PFC selection and path completion with bounded corrective feedback.
// Pattern: Diagnostic use case using the runtime parser.
// Usage: Check runs this read-free dialogue before probing Thinking controls.
// Related: internal/provider/diagnostic/check.go, internal/agent/pfc/pfc.go, internal/agent/prompt/argument.go.
package diagnostic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/pfc"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
)

func checkDialogue(ctx context.Context, client modelapi.Generator, parameters inference.Parameters, model string, thinking *bool) (modelconn.Capabilities, error) {
	caps := modelconn.Capabilities{}
	path := toolapi.Parameter{Name: "path", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}
	messages := []modelapi.Message{
		{Role: "system", Content: "This is a text-only PFC (Progressive Field Completion) protocol test. No file will be accessed and no tool will execute. In the selection phase, return one JSON object with only the tool key and the requested tool name. In the argument phase, return only the requested literal value. Do not use native tool calls, XML, code fences or explanations. JSON is used only for tool selection, never for the path value."},
		{Role: "user", Content: `Reply exactly {"tool":"read_file"}`},
	}
	for stage, expected := range []string{`{"tool":"read_file"}`, "AGENTS.md"} {
		phase := "decision"
		if stage == 1 {
			phase = "read_file.path"
		}
		for attempt := 0; attempt < 2; attempt++ {
			request := parameters.Apply(modelapi.Request{ModelID: model, Messages: messages, Thinking: thinking})
			reply, err := dialogueReply(ctx, client, request)
			if err != nil {
				return caps, fmt.Errorf("PFC diagnostic %s (max_output_tokens=%d): %w", phase, parameters.MaxOutputTokens, err)
			}
			caps.Chat, caps.Streaming = reply.done, reply.streaming
			if thinking != nil && !*thinking && reply.reasoning {
				return caps, fmt.Errorf("server emitted reasoning with Thinking OFF; verify its chat template and reasoning parser")
			}
			err = checkReply(reply, stage, path)
			if err == nil {
				messages = append(messages, modelapi.Message{Role: "assistant", Content: reply.text})
				break
			}
			if attempt == 1 {
				preview := []rune(reply.text)
				preview = preview[:min(len(preview), 160)]
				return caps, fmt.Errorf("PFC diagnostic %s failed after correction: %w; expected %q; received %q (reasoning=%t, max_output_tokens=%d). Check the serving chat template and reasoning parser", phase, err, expected, string(preview), reply.reasoning, parameters.MaxOutputTokens)
			}
			messages = append(messages,
				modelapi.Message{Role: "assistant", Content: reply.text},
				modelapi.Message{Role: "user", Content: fmt.Sprintf("PFC format correction: %s. This is a protocol test; no file access is required. Reply exactly %s, with no other text.", err, expected)},
			)
		}
		if stage == 0 {
			messages[0].Content = "Text-only protocol test; no file will be accessed.\n" + prompt.Argument("read_file", path)
			messages = append(messages, modelapi.Message{Role: "user", Content: "ARGUMENT read_file.path: Reply with only the literal path AGENTS.md, without quotes."})
		}
	}
	caps.ToolDialogue = true
	return caps, nil
}

type dialogueResponse struct {
	text                               string
	done, streaming, reasoning, native bool
}

func dialogueReply(ctx context.Context, client modelapi.Generator, request modelapi.Request) (dialogueResponse, error) {
	reply := dialogueResponse{}
	var text strings.Builder
	err := client.Generate(ctx, request, func(event modelapi.Event) error {
		if event.Retry != nil {
			text.Reset()
			reply = dialogueResponse{}
			return ctx.Err()
		}
		if text.Len()+len(event.Delta) > 1024 {
			return fmt.Errorf("PFC diagnostic reply exceeds 1024 bytes")
		}
		text.WriteString(event.Delta)
		reply.reasoning = reply.reasoning || event.Thinking != ""
		reply.native = reply.native || len(event.Calls) > 0
		if event.Done {
			reply.done, reply.streaming = true, event.Streaming
		}
		return ctx.Err()
	})
	reply.text = text.String()
	return reply, err
}

func checkReply(reply dialogueResponse, stage int, path toolapi.Parameter) error {
	if !reply.done {
		return fmt.Errorf("response ended without completion")
	}
	if reply.native {
		return fmt.Errorf("native tool calls are not PFC literal text")
	}
	if stage == 0 {
		decision, err := pfc.ParseDecision(reply.text)
		if err != nil {
			return err
		}
		if decision.Kind != pfc.Tool || decision.Name != "read_file" {
			return fmt.Errorf("expected selection of read_file")
		}
		return nil
	}
	value, err := pfc.ParseArgument(path, reply.text)
	if err != nil {
		return err
	}
	var literal string
	if err := json.Unmarshal(value, &literal); err != nil {
		return err
	}
	if literal != "AGENTS.md" {
		return fmt.Errorf("expected literal path AGENTS.md")
	}
	return nil
}
