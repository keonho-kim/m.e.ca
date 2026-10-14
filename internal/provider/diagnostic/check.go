// Purpose: Verify staged text dialogue and paired Thinking behavior through a model generator.
// Pattern: Diagnostic use case.
// Usage: Setup calls Check without executing any local tool.
// Related: internal/provider/diagnostic/control.go, internal/provider/diagnostic/effort.go.
package diagnostic

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"strings"
)

func Check(ctx context.Context, client modelapi.Generator, model string, supported, nativeOn bool, settings *inference.Settings) (modelconn.Capabilities, error) {
	caps := modelconn.Capabilities{}
	var thinking *bool
	parameters, err := inference.Resolve(settings, inference.Diagnostic)
	if err != nil {
		return caps, err
	}
	if supported {
		off := false
		thinking = &off
	}
	messages := []modelapi.Message{{Role: "system", Content: "You are testing a literal text tool pfc. Do not use native tool calls or JSON."}, {Role: "user", Content: "Reply exactly TOOL read_file"}}
	for _, expected := range []string{"TOOL read_file", "AGENTS.md"} {
		var text strings.Builder
		reasoning := false
		done := false
		err := client.Generate(ctx, parameters.Apply(modelapi.Request{ModelID: model, Messages: messages, Thinking: thinking}), func(e modelapi.Event) error {
			if e.Retry != nil {
				text.Reset()
				reasoning = false
				done = false
				return ctx.Err()
			}
			if text.Len()+len(e.Delta) > 1024 {
				return fmt.Errorf("diagnostic reply too large")
			}
			text.WriteString(e.Delta)
			reasoning = reasoning || e.Thinking != ""
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
		if thinking != nil && !*thinking && reasoning {
			return caps, fmt.Errorf("server emitted reasoning with Thinking OFF; verify its chat template and reasoning parser")
		}
		if !done || strings.TrimSpace(text.String()) != expected {
			return caps, nil
		}
		messages = append(messages, modelapi.Message{Role: "assistant", Content: text.String()}, modelapi.Message{Role: "user", Content: "ARGUMENT read_file.path: Reply with only the literal path AGENTS.md, without quotes."})
		messages[0].Content = "You are testing a literal text tool pfc.\n" + prompt.Argument("read_file", toolapi.Parameter{Name: "path", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096})
	}
	caps.ToolDialogue = true
	if supported || nativeOn {
		caps.Reasoning, err = probeThinking(ctx, client, model, "", true, settings)
		if err != nil {
			return caps, fmt.Errorf("Thinking ON diagnostic failed: %w", err)
		}
	}
	if supported && caps.Reasoning {
		reasoning, err := probeThinking(ctx, client, model, "", false, settings)
		if err != nil {
			return caps, fmt.Errorf("Thinking OFF diagnostic failed: %w", err)
		}
		if reasoning {
			return caps, fmt.Errorf("server emitted reasoning with Thinking OFF on the paired reasoning diagnostic")
		}
	}
	caps.Thinking = supported && caps.Reasoning
	return caps, nil
}
