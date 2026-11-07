// Purpose: Compose Claude whole-request bodies from the concrete dialect encoder and parent DTO boundary.
// Pattern: Request composition adapter.
// Usage: Full and cached generation paths share the same concrete Claude transformations.
// Related: internal/provider/protocol/claude/request.go, internal/provider/protocol/claude_cloud.go.
package protocol

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol/claude"
	"strings"
)

func claudeRequest(base string, request modelapi.Request, messages []Message, controls Controls) (string, []byte, string, error) {
	body, prefix, err := claudeBase(request, messages, controls)
	if err != nil {
		return "", nil, "", err
	}
	for _, message := range messages {
		digest := ""
		if len(message.continuation) > 0 {
			digest = prefix.Value()
		}
		role, blocks, err := claudeMessage(message, digest)
		if err != nil {
			return "", nil, "", err
		}
		if len(blocks) == 0 {
			continue
		}
		if err := prefix.Append(role, blocks); err != nil {
			return "", nil, "", err
		}
		last := len(body.Messages) - 1
		if last >= 0 && body.Messages[last].Role == role {
			body.Messages[last].Content = append(body.Messages[last].Content, blocks...)
		} else {
			body.Messages = append(body.Messages, claude.Message{Role: role, Content: blocks})
		}
	}
	encoded, err := wireJSON.Marshal(body)
	return base + "/messages", encoded, prefix.Value(), err
}

func claudeBase(request modelapi.Request, messages []Message, controls Controls) (claude.Request, *claude.Prefix, error) {
	var system []string
	for _, message := range messages {
		if message.Role == "system" {
			text, _ := message.Content.(string)
			system = append(system, text)
		}
	}
	return claude.Base(request, strings.Join(system, "\n\n"), controls.Think, controls.ReasoningEffort)
}
func claudeMessage(message Message, prefix string) (string, []claude.Block, error) {
	input, err := nativeMessage(message)
	if err != nil {
		return "", nil, err
	}
	return claude.EncodeMessage(input, prefix)
}
func claudeContinuation(prefix string) func([]byte) ([]byte, error) {
	return func(raw []byte) ([]byte, error) { return claude.Bind(raw, prefix) }
}
