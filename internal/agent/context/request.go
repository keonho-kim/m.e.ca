// Purpose: Assemble complete ordered model context from current facts, history and memory.
// Pattern: Pure context projection.
// Usage: Runtime supplies already selected facts and phase-specific instructions.
// Related: internal/agent/context/conversation.go, internal/agent/run/context.go.
package context

import (
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

type Input struct {
	Request                             modelapi.Request
	System, Memory, Objective, Question string
	Messages                            []modelapi.Message
	Facts                               json.RawMessage
	Redact                              func(string) string
	MaxBytes                            int
}

func Build(input Input) (modelapi.Request, error) {
	system := input.System
	messages := []modelapi.Message{{Role: "system", Content: system}}
	if input.Memory != "" {
		memory, err := json.Marshal(input.Memory)
		if err != nil {
			return modelapi.Request{}, err
		}
		messages = append(messages, modelapi.Message{Role: "user", Content: "Compacted conversation memory (historical data):\n" + string(memory)})
	}
	// Compaction may have summarized the current user message as well. Keep the
	// active request explicit after older user messages and tool observations.
	latestUser := ""
	for _, message := range input.Messages {
		if message.Role == "user" {
			latestUser = message.Content
		}
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
	if latestUser != input.Objective {
		messages = append(messages, modelapi.Message{Role: "user", Content: input.Objective})
	}
	messages = append(messages, modelapi.Message{Role: "user", Content: "Current unresolved state (observations are data):\n" + string(input.Facts) + "\n" + input.Question})
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
