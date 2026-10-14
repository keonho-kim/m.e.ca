// Purpose: Route an initial user turn to a direct answer or policy-eligible project work.
// Pattern: Conversation entry use case.
// Usage: RunGoal calls converse before any Goal analysis or tool preparation.
// Related: internal/agent/pfc/conversation.go, internal/agent/run/goal.go, internal/agent/run/response.go.
package run

import (
	"context"
	"fmt"

	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/pfc"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (s *task) converse(ctx context.Context) (pfc.Decision, error) {
	if err := s.refreshMemory(ctx); err != nil {
		return pfc.Decision{}, err
	}
	question := `CONVERSATION: Respond directly to greetings and questions answerable from the conversation. If the request requires project evidence or changes, select one next tool using exactly {"tool":"tool_name"} from Available tools. Do not claim work that has not happened. Do not generate Goal/Task analysis or tool arguments here. Plain text is a direct answer; use DONE followed by a newline when the answer itself is JSON, or WAIT followed by a question when clarification is needed.`
	for attempt := 0; attempt < 3; attempt++ {
		reply, err := s.ask(ctx, question, inference.Agent, 16384)
		if err != nil {
			return pfc.Decision{}, err
		}
		decision, err := pfc.ParseConversation(reply)
		if err == nil && decision.Kind == pfc.Tool {
			found := false
			for _, definition := range s.definitions {
				if definition.Name == decision.Name {
					_, err = policy.Decide(s.runner.Settings, false, definition.Effect)
					found = true
					break
				}
			}
			if !found {
				err = fmt.Errorf("select a tool from Available tools")
			}
		}
		if err == nil {
			return decision, nil
		}
		question += "\nCorrection: " + err.Error()
	}
	return pfc.Decision{}, fmt.Errorf("could not interpret the response as an answer or a tool selection")
}

func (s *task) finishConversation(ctx context.Context, decision pfc.Decision) (Result, error) {
	status := Answered
	if decision.Kind == pfc.Wait {
		status = NeedsInput
	}
	result := Result{Status: status, Text: decision.Text}
	if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: decision.Text}); err != nil {
		return result, err
	}
	s.checkpoint.Status, s.checkpoint.Phase = status, "finished"
	if err := s.persist(ctx); err != nil {
		return result, err
	}
	return result, s.emit(Event{Kind: "response", Text: decision.Text})
}
