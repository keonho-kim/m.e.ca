// Purpose: Assemble one main-agent context with current unresolved execution facts.
// Pattern: Context preparation use case.
// Usage: The single run loop requests native tools and complete uncompacted history.
// Related: internal/agent/context/request.go, internal/agent/run/run.go.
package run

import (
	"encoding/json"
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
func (s *execution) request(question string, role inference.Role) (modelapi.Request, error) {
	parameters, err := inference.Resolve(s.runner.Inference, role)
	if err != nil {
		return modelapi.Request{}, err
	}
	var defs []toolapi.Definition
	for _, d := range s.definitions {
		if s.allowTool(d.Effect) == nil {
			defs = append(defs, toolapi.WithIntent(d))
		}
	}
	if s.planning {
		defs = append(defs, planDefinition())
	}
	if s.changed {
		question += "\nReport completed changes, actual checks and observations, and anything unverified. Use recorded outcomes; do not infer overall correctness from a successful command. If the request is satisfied, answer rather than polishing speculatively."
	}
	facts := struct {
		Blocker       string `json:"blocker,omitempty"`
		PendingReview bool   `json:"pending_review,omitempty"`
		Observation   string `json:"observation,omitempty"`
	}{s.checkpoint.Blocker, s.checkpoint.PendingReview, s.checkpoint.Observation}
	body, err := json.Marshal(facts)
	if err != nil {
		return modelapi.Request{}, err
	}
	if s.runner.PreviousRun != nil && (s.runner.PreviousRun.Status == Interrupted || s.runner.PreviousRun.Status == Stopped || s.runner.PreviousRun.Status == Unverified) {
		previous := *s.runner.PreviousRun
		previous.Arguments = nil
		previous.Grounding = nil
		previous.Conclave = nil
		encoded, encodeErr := json.Marshal(previous)
		if encodeErr != nil {
			return modelapi.Request{}, encodeErr
		}
		question += "\nInterrupted prior run (historical data; inspect current state, never replay drafts): " + string(encoded)
	}
	return contextbuild.Build(contextbuild.Input{Request: parameters.Apply(modelapi.Request{ModelID: s.runner.Model, Thinking: s.runner.Thinking, ReasoningEffort: s.runner.ReasoningEffort, Tools: defs}), System: s.system, Memory: s.memoryContext, Objective: s.checkpoint.Objective, Question: question, Messages: s.messages, Facts: body, Redact: s.runner.Redact, MaxBytes: s.limits.MaxContextBytes})
}
