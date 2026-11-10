// Purpose: Filter selected response bodies while preserving global controls and resumable cursors.
// Pattern: SSE subscription projection at the transport boundary.
// Usage: The SSE handler borrows the selected workspace and prepares reset snapshots.
// Related: internal/serverapi/http.go, internal/commands/server/events_read.go.
package serverapi

import (
	"context"
	"github.com/keonho-kim/meca/internal/commands/server"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"net/url"
)

type bodyScope struct {
	workspace, session  string
	scoped, sessionOnly bool
}

func globalEvent(kind string) bool {
	switch kind {
	case "request_started", "request_finished", "draft_reset", "session_deleted", "session_opened", "session_closed", "settings_changed", "approval_required", "approval_closed", "usage", "usage_progress":
		return true
	default:
		return false
	}
}

func (t *Transport) observerService(ctx context.Context, query url.Values) (*server.Service, bodyScope, func(), error) {
	scope := bodyScope{workspace: query.Get("workspace_id"), session: query.Get("session_id"), scoped: query.Has("workspace_id") || query.Has("session_id"), sessionOnly: query.Has("session_id")}
	if len(scope.workspace) > 128 || len(scope.session) > 128 || scope.session != "" && !sessionstate.ValidID(scope.session) {
		return nil, scope, nil, fault.New(diag.InvalidArgument, diag.ServerPageInvalid)
	}
	if scope.workspace == "" {
		scope.workspace = t.api.ScopeID
	}
	if !scope.scoped || scope.workspace == t.api.ScopeID {
		return t.api.Service, scope, func() {}, nil
	}
	if t.api.Workspaces == nil {
		return nil, scope, nil, fault.New(diag.NotFound, diag.UIWorkspaceUnknown)
	}
	service, release, err := t.api.Workspaces.Acquire(ctx, scope.workspace)
	return service, scope, release, err
}
func (scope bodyScope) prepare(ctx context.Context, service *server.Service, batch server.Batch, task string) (server.Batch, error) {
	if scope.scoped && batch.Snapshot != nil {
		state, err := service.StatePage(ctx, scope.session, task, "", 32)
		if err != nil {
			return server.Batch{}, err
		}
		batch.Snapshot = &state
	}
	return batch, nil
}
func (scope bodyScope) filter(batch server.Batch) server.Batch {
	if !scope.scoped || batch.Snapshot != nil {
		return batch
	}
	filtered := make([]server.Event, 0, len(batch.Events))
	for _, event := range batch.Events {
		if !globalEvent(event.Record.Kind) && (event.Workspace != scope.workspace || scope.sessionOnly && (scope.session == "" || event.Session != scope.session)) {
			continue
		}
		filtered = append(filtered, event)
	}
	return server.Batch{Events: filtered}
}
