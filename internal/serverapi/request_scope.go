// Purpose: Resolve authenticated binary/file HTTP requests to their explicit project runtime.
// Pattern: Workspace lease at the transport boundary.
// Usage: Attachment, image, task and file-stream handlers borrow a scoped API.
// Related: internal/serverapi/application.go, internal/commands/server/workspaces.go.
package serverapi

import (
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"net/http"
)

func (t *Transport) requestAPI(r *http.Request) (*API, func(), error) {
	id := r.Header.Get("X-Meca-Workspace")
	if len(id) > 128 || len(r.Header.Values("X-Meca-Workspace")) > 1 {
		return nil, nil, fault.New(diag.InvalidArgument, diag.ServerPageInvalid)
	}
	if id == "" {
		return t.api, func() {}, nil
	}
	if t.api.Workspaces == nil {
		return nil, nil, fault.New(diag.NotFound, diag.UIWorkspaceUnknown)
	}
	service, done, err := t.api.Workspaces.Acquire(r.Context(), id)
	if err != nil {
		return nil, nil, err
	}
	return t.api.scoped(service, id), done, nil
}
