// Purpose: Accept an explicit native Plan proposal only in read-only Plan mode.
// Pattern: Native control transition.
// Usage: action routes submit_plan here; DraftPlan persists the validated revision.
// Related: internal/agent/plan/proposal.go, internal/agent/run/planning.go.
package run

import (
	"context"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func planDefinition() toolapi.Definition {
	return toolapi.WithIntent(toolapi.Definition{Name: "submit_plan", Description: "Submit a complete read-only plan for user approval. This saves a draft, never executes its steps.", Parameters: []toolapi.Parameter{
		{Name: "objective", Description: "User-facing objective.", Kind: toolapi.Text, Required: true, MaxBytes: 16384},
		{Name: "steps", Description: "Ordered actionable steps, including verification.", Kind: toolapi.TextList, Required: true, MaxBytes: 4096, Max: 50},
	}})
}
func (s *execution) submitPlan(ctx context.Context, reply modelapi.Response) error {
	if !s.planning {
		return s.returnToolFeedback(ctx, reply, "submit_plan is available only in Plan mode")
	}
	if _, err := toolapi.ValidateArguments(planDefinition(), reply.Calls[0].Arguments); err != nil {
		return s.returnToolFeedback(ctx, reply, err.Error())
	}
	call, reason, err := toolapi.ExtractIntent(reply.Calls[0])
	if err != nil {
		return s.returnToolFeedback(ctx, reply, err.Error())
	}
	if _, err := planning.ParsePlanProposal(string(call.Arguments)); err != nil {
		return s.returnToolFeedback(ctx, reply, err.Error())
	}
	if err := s.ground(ctx, call, reason); err != nil {
		return err
	}
	if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: reply.Text, Reasoning: reply.Reasoning, Calls: reply.Calls}); err != nil {
		return err
	}
	if err := s.save(ctx, modelapi.Message{Role: "tool", ToolCallID: call.ID, ToolName: call.Name, Content: `{"accepted":true,"execution":"awaiting user approval"}`}); err != nil {
		return err
	}
	s.result.Text = string(call.Arguments)
	s.checkpoint.Status = PlanReady
	return nil
}
