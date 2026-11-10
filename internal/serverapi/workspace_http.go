// Purpose: Bind stored image and task URLs to a project without putting credentials in URLs.
// Pattern: Namespace routing adapter.
// Usage: Browser images use /v1/workspaces/{id}/sessions/... with the UI cookie.
// Related: internal/serverapi/request_scope.go, internal/serverapi/history_image.go.
package serverapi

import (
	"net/http"
	"strings"
)

func (t *Transport) workspaceHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/v1/workspaces/"), "/", 2)
	if len(parts) != 2 || parts[0] == "" || len(parts[0]) > 128 {
		http.NotFound(w, r)
		return
	}
	r = r.Clone(r.Context())
	r.Header.Set("X-Meca-Workspace", parts[0])
	r.URL.Path = "/v1/" + parts[1]
	r.URL.RawPath = ""
	switch {
	case strings.HasPrefix(r.URL.Path, "/v1/sessions/"):
		t.historyImage(w, r)
	case strings.HasPrefix(r.URL.Path, "/v1/tasks/"):
		t.taskControl(w, r)
	default:
		http.NotFound(w, r)
	}
}
