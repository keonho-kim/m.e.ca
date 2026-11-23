// Purpose: Measure invitation phases using bounded diagnostic metadata.
// Pattern: Bounded diagnostic events.
// Usage: Invitation admission, private preparation, activation and the first model call report elapsed milliseconds.
// Related: internal/agent/team/invitation.go, internal/agent/team/peer.go, internal/diagnostics/catalog/events.go.
package team

import (
	"context"
	"time"

	"github.com/keonho-kim/meca/internal/diagnostics"
	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func (r *Room) logInvitationPhase(ctx context.Context, id uint64, phase string, started time.Time) {
	diagnostics.Log(ctx, catalog.TeamInvitationPhase, "component", "team", "run_id", r.runID,
		"invitation_id", id, "operation", phase, "duration_ms", time.Since(started).Milliseconds())
}
