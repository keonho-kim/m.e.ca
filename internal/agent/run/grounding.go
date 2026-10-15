// Purpose: Persist the native call's intent before any tool is prepared.
// Pattern: Mandatory execution transition without model I/O.
// Usage: action supplies validated intent and exact executable arguments.
// Related: internal/agent/grounding/record.go, internal/agent/run/action.go.
package run

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/grounding"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func (s *execution) ground(ctx context.Context, call toolapi.Call, reason string) error {
	record, err := grounding.Bind(call, reason, s.checkpoint.ID)
	if err != nil {
		return err
	}
	if err := s.record(ctx, Event{Kind: "self_grounding", Grounding: &record}); err != nil {
		return err
	}
	s.checkpoint.Grounding = &record
	return s.persist(ctx)
}
