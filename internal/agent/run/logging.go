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
	if e.Generation != nil {
		fields = append(fields, "generation", e.Generation.ID, "attempt", e.Generation.Attempt, "duration_ms", e.Generation.ElapsedMillis)
	}
	if e.Call != nil {
		fields = append(fields, "operation", e.Call.Name)
	}
	if e.Skill != nil {
		fields = append(fields, "skill_id", e.Skill.ID, "skill", e.Skill.Name)
	}
	diagnostics.Log(ctx, event, fields...)
}
