// Purpose: Assemble each complete model request into one owned message array.
// Pattern: Pure context projection with an execution-owned reuse boundary.
// Usage: Runtime supplies retained history, instructions and the active request.
// Related: internal/agent/context/projection.go, internal/agent/context/instructions.go.
package context

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Input struct {
	ImageInput                bool
	Request                   modelapi.Request
	System, Memory, Objective string
	ProjectRoot               string
	Observations              string
	Messages                  []modelapi.Message
	ClientContext             []modelapi.Message
	SkillReads                []modelapi.Message
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
	messages := make([]modelapi.Message, 0, len(input.Messages)+len(input.ClientContext)+len(input.SkillReads)+6)
	messages = append(messages, modelapi.Message{Role: "system", Content: p.instructions(input)})
	if input.Memory != "" {
		messages = append(messages, modelapi.Message{Role: "user", Content: p.summary(input)})
	} else {
		p.memory = summarySnapshot{}
	}
	present := input.Objective == ""
	for i := len(input.Messages) - 1; i >= 0 && !present; i-- {
		present = input.Messages[i].Role == "user" && input.Messages[i].Content == input.Objective
	}
	if !present && input.Memory != "" {
		messages = append(messages, modelapi.Message{Role: "user", Content: input.Redact(input.Objective)})
		present = true
	}
	if len(input.ClientContext) > 0 {
		if p.client == nil {
			p.client = &Projection{}
		}
		clientInput := input
		clientInput.Messages = input.ClientContext
		var err error
		messages, _, err = p.client.appendHistory(clientInput, messages)
		if err != nil {
			return modelapi.Request{}, err
		}
	} else {
		p.client = nil
	}
	prefix := len(messages)
	messages, bytes, err := p.appendHistory(input, messages)
	if err != nil {
		return modelapi.Request{}, err
	}
	var skillBytes int
	messages, skillBytes = p.appendSkillReads(input, messages)
	bytes += skillBytes
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
	if input.Observations != "" {
		message := modelapi.Message{Role: "user", Content: p.observations(input)}
		messages = append(messages, message)
		bytes += message.ContextBytes()
	} else {
		p.work = summarySnapshot{}
	}
	if bytes > input.MaxBytes {
		return modelapi.Request{}, faults.Errorf(diagcatalog.ContextFullConversationContextExceedsBytesNoMessages, input.MaxBytes)
	}
	input.Request.Messages = messages
	return input.Request, nil
}
