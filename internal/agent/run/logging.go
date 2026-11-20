// Purpose: Record execution transitions as small diagnostics without logging model or tool bodies.
// Pattern: Runtime observation adapter.
// Usage: The serial emit boundary calls logEvent only after successful event delivery.
// Related: internal/agent/run/emitter.go, internal/diagnostics/logger.go.
package run

import (
	"context"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func logEvent(ctx context.Context, e runrecord.Event, session string) {
	event, registered := diagcatalog.RuntimeEvent(e.PublicKind())
	if !registered {
		return
	}
	fields := []any{"component", "agent", "session_id", session, "run_id", e.RunID, "call_id", e.CallID, "error_code", e.ErrorCode}
	if g := e.Generation; g != nil {
		fields = append(fields, "generation", g.ID, "attempt", g.Attempt, "duration_ms", g.ElapsedMillis, "role", string(g.Role))
		if g.Context != nil {
			fields = append(fields, "input_tokens", g.Context.Input, "input_source", g.Context.Source, "context_window", g.Context.Window, "output_budget", g.Context.OutputBudget)
		}
		if g.BeforeCompaction != nil {
			fields = append(fields, "before_compact_tokens", g.BeforeCompaction.Input, "before_compact_source", g.BeforeCompaction.Source)
		}
		if g.FirstOutputMillis != nil {
			fields = append(fields, "first_output_ms", *g.FirstOutputMillis)
		}
		if g.Speed != nil {
			fields = append(fields, "rate_source", g.Speed.Source, "timed_tokens", g.Speed.Output, "timed_ns", g.Speed.Duration.Nanoseconds())
		}
	}
	if e.Call != nil {
		fields = append(fields, "operation", e.Call.Name)
	}
	if e.Skill != nil {
		fields = append(fields, "skill_id", e.Skill.ID, "skill", e.Skill.Name)
	}
	diagnostics.Log(ctx, event, fields...)
}
