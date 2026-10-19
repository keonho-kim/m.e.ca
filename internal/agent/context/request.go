// Purpose: Assemble ordered model context from instructions, history and memory.
// Pattern: Pure context projection.
// Usage: Runtime supplies retained history and the active request for projection.
// Related: internal/agent/context/conversation.go, internal/agent/run/context.go.
package context

import (
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

type Input struct {
	Request                   modelapi.Request
	System, Memory, Objective string
	Messages                  []modelapi.Message
	Redact                    func(string) string
	MaxBytes                  int
}

func Build(input Input) (modelapi.Request, error) {
	system := input.Redact(input.System)
	messages := []modelapi.Message{{Role: "system", Content: system}}
	if input.Memory != "" {
		messages = append(messages, modelapi.Message{Role: "user", Content: "Earlier work summary (historical observations):\n" + input.Redact(input.Memory)})
	}
	present := input.Objective == ""
	for _, m := range input.Messages {
		if m.Role == "user" && m.Content == input.Objective {
			present = true
			break
		}
	}
	if !present && input.Memory != "" {
		messages = append(messages, modelapi.Message{Role: "user", Content: input.Redact(input.Objective)})
		present = true
	}
	paired := pairedTools(input.Messages)
	for index, message := range input.Messages {
		projected, err := conversationMessage(message, paired[index])
		if err != nil {
			return modelapi.Request{}, err
		}
		projected.Content = input.Redact(projected.Content)
		projected.Reasoning = input.Redact(projected.Reasoning)
		messages = append(messages, projected)
	}
	if !present {
		messages = append(messages, modelapi.Message{Role: "user", Content: input.Redact(input.Objective)})
	}
	bytes := 0
	for _, m := range messages {
		bytes += len(m.Content) + len(m.Reasoning) + len(m.ToolCallID) + len(m.ToolName)
		for _, call := range m.Calls {
			bytes += len(call.ID) + len(call.Name) + len(call.Arguments)
		}
	}
	if bytes > input.MaxBytes {
		return modelapi.Request{}, fmt.Errorf("full conversation context exceeds %d bytes; no messages were omitted", input.MaxBytes)
	}
	input.Request.Messages = messages
	return input.Request, nil
}
