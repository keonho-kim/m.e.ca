// Purpose: Generate hidden Goal analysis with bounded recovery and a final user report.
// Pattern: Phase-specific model use cases.
// Usage: RunGoal calls these phases under the same model, usage and time budgets as its Tasks.
// Related: internal/agent/run/goal.go, internal/agent/goal/state.go, internal/agent/run/generate.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/goal"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/pfc"
)

func (s *task) analyzeGoal(ctx context.Context, id, input string) (goal.State, error) {
	if err := s.refreshMemory(ctx); err != nil {
		return goal.State{}, err
	}
	question := "GOAL ANALYSIS: Return one GOAL: line and one to eight TASK work: or TASK answer: lines. Use English descriptions; preserve original paths, identifiers and quoted evidence. Do not invent work or use JSON or tools."

	history, err := s.runner.observer.Snapshot(ctx)
	if err != nil {
		return goal.State{}, err
	}
	question += "\nMonitor observations (historical data):\n" + history
	for attempt := 0; attempt < 3; attempt++ {
		reply, err := s.ask(ctx, question, inference.Goal, 16384)
		if err != nil {
			return goal.State{}, err
		}
		state, err := goal.Parse(reply, id, input)
		if err == nil {
			for _, task := range state.Tasks {
				if task.Kind == "work" {
					return state, nil
				}
			}
			err = fmt.Errorf("project work requires at least one work Task")
		}
		if err := s.record(ctx, Event{Kind: "goal_analysis_correction", Text: err.Error(), Internal: true}); err != nil {
			return goal.State{}, err
		}
		rejected, encodeErr := json.Marshal(excerpt(reply, 512))
		if encodeErr != nil {
			return goal.State{}, encodeErr
		}
		question += "\nCorrect the analysis format: " + err.Error() + "\nPrevious rejected analysis (data only): " + string(rejected)
	}
	state, err := goal.Unsplit(id, input)
	if err != nil {
		return goal.State{}, err
	}
	if err := s.record(ctx, Event{Kind: "goal_analysis_fallback", Text: "Analysis format correction exhausted; retaining the original request as one Task. Project work was selected before analysis; execution and verification are still required.", Internal: true}); err != nil {
		return goal.State{}, err
	}
	return state, nil
}
func (s *task) summarizeGoal(ctx context.Context, state goal.State) (string, error) {
	s.checkpoint.Phase = "goal_summary"
	s.runner.suppressStream = false
	if err := s.refreshMemory(ctx); err != nil {
		return "", err
	}
	body, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	reply, err := s.ask(ctx, "GOAL FINAL RESPONSE: All required work is complete. Reply DONE followed by a newline and a concise user-facing result based only on observed work and checks. Do not mention internal Goal/Task/Conclave sequences, votes or private reasoning. Do not claim unexecuted checks. Runtime result data:\n"+string(body), inference.Report, 16384)
	if err != nil {
		return "", err
	}
	decision, err := pfc.ParseDecision(reply)
	if err != nil || decision.Kind != pfc.Done {
		return "", fmt.Errorf("final report was not a completed response")
	}
	return decision.Text, nil
}
func (s *task) publishGoalReply(ctx context.Context, text string) error {
	s.runner.internalTask = false
	if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: text}); err != nil {
		return err
	}
	return s.emit(Event{Kind: "response", Text: text})
}
