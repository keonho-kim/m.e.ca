// Purpose: Encode Claude request fields and preserve signed continuation blocks at the dialect boundary.
// Pattern: Concrete Messages API request adapter.
// Usage: Full and cached parent encoders call Base and EncodeMessage with immutable native inputs.
// Related: internal/provider/protocol/claude/continuation.go, internal/provider/protocol/claude/prefix.go.
package claude

import (
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
)

func Base(request modelapi.Request, system string, thinking any, effort string) (Request, *Prefix, error) {
	body := Request{Model: request.ModelID, MaxTokens: request.MaxOutput, Stream: true, Thinking: thinking}
	if effort != "" {
		body.OutputConfig = map[string]string{"effort": effort}
	}
	for _, d := range request.Tools {
		body.Tools = append(body.Tools, Tool{Name: d.Name, Description: d.Description, Schema: d.JSONSchema()})
	}
	body.System = system
	prefix, err := NewPrefix(body.System, body.Tools)
	return body, prefix, err
}

func EncodeMessage(message native.Input, prefix string) (string, []Block, error) {
	if message.Role == "system" {
		return "", nil, nil
	}
	text := message.Text
	if message.Role == "tool" {
		return "user", []Block{{Type: "tool_result", ToolUseID: message.ToolCallID, Content: text}}, nil
	}
	blocks, prior, err := messageBlocks(message)
	if err != nil {
		return "", nil, err
	}
	if prior != "" && prior != prefix {
		// Thinking is bound to the earlier conversation, while completed text and tools remain observations.
		kept := blocks[:0]
		for _, block := range blocks {
			if block.Type != "thinking" && block.Type != "redacted_thinking" {
				kept = append(kept, block)
			}
		}
		clear(blocks[len(kept):])
		blocks = kept
	}
	return message.Role, blocks, nil
}

func messageBlocks(message native.Input) ([]Block, string, error) {
	calls := message.Calls
	text := message.Text
	var blocks []Block
	if len(message.State) > 0 {
		prior, err := Restore(message.State)
		if err != nil {
			return nil, "", err
		}
		blocks = prior.Blocks
		callIndex := 0
		var original strings.Builder
		for i := range blocks {
			block := &blocks[i]
			switch block.Type {
			case "text":
				original.WriteString(block.Text)
			case "tool_use":
				if callIndex >= len(calls) || calls[callIndex].ID != block.ID || calls[callIndex].Name != block.Name {
					return nil, "", faults.Errorf(diagcatalog.ClaudeClaudeContinuationDiffersFromToolHistory)
				}
				if !native.SameArguments(string(block.Input), string(calls[callIndex].Arguments)) {
					return nil, "", faults.Errorf(diagcatalog.ClaudeClaudeContinuationArgumentsDifferFromToolHistory)
				}
				callIndex++
			case "thinking", "redacted_thinking":
			default:
				return nil, "", faults.Errorf(diagcatalog.ClaudeUnsupportedClaudeContinuationBlock)
			}
		}
		if callIndex != len(calls) {
			return nil, "", faults.Errorf(diagcatalog.ClaudeClaudeContinuationOmittedAToolCall)
		}
		if original.String() != text && strings.TrimSpace(original.String()) != text {
			return nil, "", faults.Errorf(diagcatalog.ClaudeClaudeContinuationTextDiffersFromMessageHistory)
		}
		return blocks, prior.Prefix, nil
	}
	if text != "" {
		blocks = append(blocks, Block{Type: "text", Text: text})
	}
	if len(message.Parts) > 0 {
		for _, part := range message.Parts {
			if part.ImageURL == nil {
				blocks = append(blocks, Block{Type: "text", Text: part.Text})
				continue
			}
			mime, encoded, ok := native.DataImage(part.ImageURL.URL)
			if !ok {
				return nil, "", faults.Errorf(diagcatalog.ClaudeClaudeRequiresEmbeddedImageData)
			}
			blocks = append(blocks, Block{Type: "image", Source: &ImageSource{Type: "base64", Media: mime, Data: encoded}})
		}
	}
	for _, call := range calls {
		blocks = append(blocks, Block{Type: "tool_use", ID: call.ID, Name: call.Name, Input: call.Arguments})
	}
	return blocks, "", nil
}
