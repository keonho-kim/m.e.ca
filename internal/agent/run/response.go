// Purpose: Select only eligible user-facing response text from an unfinished decision.
// Pattern: Presentation projection.
// Usage: Generation uses responseBody to keep internal sequence output hidden.
// Related: internal/agent/run/generate.go, internal/agent/pfc/pfc.go.
package run

import "github.com/keonho-kim/v-code/internal/agent/pfc"

// Only a user-facing body can be projected before the decision finishes.
func (s *task) responseBody(text string) string {
	if s.runner.suppressStream || s.planning {
		return ""
	}
	if s.checkpoint.Phase == "goal_summary" {
		decision, err := pfc.ParseDecision(text)
		if err == nil && decision.Kind == pfc.Done {
			return decision.Text
		}
		return ""
	}
	if s.checkpoint.Phase != "decision" {
		return ""
	}
	decision, err := pfc.ParseDecision(text)
	if err != nil {
		return ""
	}
	switch decision.Kind {
	case pfc.Wait:
		return decision.Text
	case pfc.Done:
		if !s.checkpoint.PendingReview && (!s.worked && (s.runner.assignment == nil || s.runner.assignment.Kind == "answer") || s.checkpoint.Evidence != "") {
			return decision.Text
		}
	}
	return ""
}
