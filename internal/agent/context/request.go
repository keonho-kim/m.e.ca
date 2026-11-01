// Purpose: Assemble each complete model request into one owned message array.
// Pattern: Pure context projection with an execution-owned reuse boundary.
// Usage: Runtime supplies retained history, instructions and the active request.
// Related: internal/agent/context/projection.go, internal/agent/context/instructions.go.
package context

import (
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type Input struct {
	ImageInput                bool
	Request                   modelapi.Request
	System, Memory, Objective string
	ProjectRoot               string
	Messages                  []modelapi.Message
	Redact                    func(string) string
	MaxBytes                  int
	Projection                *Projection
	RedactionVersion          func() int
}

func Build(input Input) (modelapi.Request, error) {
	if input.Projection != nil && input.RedactionVersion != nil {
		return input.Projection.build(input)
	}
	var uncached Projection
	input.RedactionVersion = func() int { return 0 }
	return uncached.build(input)
}

func (p *Projection) buildOnce(input Input) (modelapi.Request, error) {
	// One request owns these headers. Reusing this backing array across model
	// calls would mutate earlier requests retained by a provider or observer.
	messages := make([]modelapi.Message, 0, len(input.Messages)+4)
	messages = append(messages, modelapi.Message{Role: "system", Content: p.instructions(input)})
	if input.Memory != "" {
		messages = append(messages, modelapi.Message{Role: "user", Content: "Earlier work summary (historical observations):\n" + input.Redact(input.Memory)})
	}
	present := input.Objective == ""
	for i := len(input.Messages) - 1; i >= 0 && !present; i-- {
		present = input.Messages[i].Role == "user" && input.Messages[i].Content == input.Objective
	}
	if !present && input.Memory != "" {
		messages = append(messages, modelapi.Message{Role: "user", Content: input.Redact(input.Objective)})
		present = true
	}
	prefix := len(messages)
	messages, bytes, err := p.appendHistory(input, messages)
	if err != nil {
		return modelapi.Request{}, err
	}
	for _, message := range messages[:prefix] {
		bytes += message.ContextBytes()
	}
	if !input.ImageInput {
		for i := prefix; i < len(messages); i++ {
			if len(messages[i].Images) > 0 {
				bytes -= messages[i].ContextBytes()
				messages[i].Content += modelapi.ImageOmissionNotice
				messages[i].Images = nil
				bytes += messages[i].ContextBytes()
			}
		}
	}
	if !present {
		message := modelapi.Message{Role: "user", Content: input.Redact(input.Objective)}
		messages = append(messages, message)
		bytes += message.ContextBytes()
	}
	if bytes > input.MaxBytes {
		return modelapi.Request{}, fmt.Errorf("full conversation context exceeds %d bytes; no messages were omitted", input.MaxBytes)
	}
	input.Request.Messages = messages
	return input.Request, nil
}
