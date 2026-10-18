// Purpose: Assemble the main agent's context from its request, constraints and observed facts.
// Pattern: Context preparation use case.
// Usage: The main loop supplies its direct tools and single conversation.
// Related: internal/agent/context/request.go, internal/agent/run/run.go.
package run

import (
	contextbuild "github.com/keonho-kim/v-code/internal/agent/context"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"unicode/utf8"
)

func excerpt(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	end := limit
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + "\n[excerpt; full record retained]"
}
func (s *execution) request(question string) (modelapi.Request, error) {
	parameters, err := inference.Resolve(s.runner.Inference, inference.Agent)
	if err != nil {
		return modelapi.Request{}, err
	}
	var defs []toolapi.Definition
	for _, d := range s.definitions {
		defs = append(defs, toolapi.WithIntent(d))
	}

	body, err := s.taskContext()
	if err != nil {
		return modelapi.Request{}, err
	}
	return contextbuild.Build(contextbuild.Input{Request: parameters.Apply(modelapi.Request{ModelID: s.runner.Model, Thinking: s.runner.Thinking, ReasoningEffort: s.runner.ReasoningEffort, Tools: defs}), System: s.system, Memory: s.memoryContext, Objective: s.checkpoint.Objective, Question: question, Messages: s.messages, Facts: body, Redact: s.runner.Redact, MaxBytes: s.limits.MaxContextBytes})
}
