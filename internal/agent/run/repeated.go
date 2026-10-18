// Purpose: Recognize completed mutations against current versions without replaying effects.
// Pattern: Version-bound result reuse.
// Usage: Native dispatch checks repeats after grounding and pairs reused observations with the new call.
// Related: internal/agent/run/call_validation.go, internal/agent/run/observation.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/policy"
	"maps"
	"slices"
)

type mutationReceipt struct {
	callID   string
	versions map[string]string
	success  bool
}

func (s *execution) repeated(ctx context.Context, v validatedCall) (*mutationReceipt, error) {
	signature := callSignature(v.call)
	if revision, exists := s.commands[signature]; v.definition.Effect == policy.Command && exists && revision == s.checkpoint.ChangeRevision {
		return nil, fmt.Errorf("Repeated identical command blocked; it already ran without an intervening operation. Reuse its recorded command, exit status and output; it was not executed again.")
	}
	prior, exists := s.mutations[signature]
	if v.definition.Effect != policy.Change || !exists {
		return nil, nil
	}
	reader, ok := s.runner.Tools.(fileVersions)
	if !ok || len(prior.versions) == 0 || !prior.success {
		return nil, fmt.Errorf("Previous call %s may have applied effects. Nothing replayed. Inspect its actual outcome and current target before submitting only the remaining change.", prior.callID)
	}
	current, err := reader.FileVersions(ctx, s.runner.Settings, slices.Collect(maps.Keys(prior.versions)))
	if err != nil {
		return nil, err
	}
	for path, version := range prior.versions {
		if version == "" || current[path] != version {
			return nil, nil
		}
	}
	return &prior, nil
}

func (s *execution) reuseMutation(ctx context.Context, v validatedCall, prior mutationReceipt) error {
	body, err := json.Marshal(struct {
		Executed     bool              `json:"executed"`
		Status       string            `json:"status"`
		SourceCallID string            `json:"source_call_id"`
		Versions     map[string]string `json:"versions"`
	}{false, "already_applied", prior.callID, prior.versions})
	if err != nil {
		return err
	}
	if err := s.save(ctx, modelapi.Message{Role: "tool", ToolCallID: v.call.ID, ToolName: v.call.Name, Content: string(body)}); err != nil {
		return err
	}
	s.checkpoint.Observation = "This exact change is already applied and its target versions still match. Nothing executed again; preserve it and address only remaining work."
	event := Event{Kind: "tool_reused", CallID: v.call.ID, Call: &v.call, Text: s.checkpoint.Observation, Output: body}
	if err := s.record(ctx, event); err != nil {
		return err
	}
	return s.persist(ctx)
}
