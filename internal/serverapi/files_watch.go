// Purpose: Serve the same cancellable file-change stream over native HTTP and Protobuf RPC.
// Pattern: Streaming transport adapter.
// Usage: The browser opens GET /v1/files/events; RPC clients use WatchFiles.
// Related: internal/commands/server/file_watch.go, internal/serverapi/rpc.go.
package serverapi

import (
	"context"
	"net/http"
	"net/url"

	"connectrpc.com/connect/v2"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/serverapi/admission"
	"github.com/keonho-kim/meca/internal/serverapi/eventstream"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"github.com/keonho-kim/meca/internal/serverapi/gen/mecav1/mecav1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

func (a *API) WatchFiles(ctx context.Context, r *v1.FileWatchRequest, stream mecav1connect.MecaServiceWatchFilesServerStream) error {
	if len(r.GetPath()) > 4096 || len(r.GetWorkspaceId()) > 128 {
		return connect.NewError(connect.CodeInvalidArgument, diag.Text(diag.ServerPageInvalid))
	}
	admission.ReleaseInput(ctx)
	api := a
	if a.Workspaces != nil && r.GetWorkspaceId() != "" {
		s, done, err := a.Workspaces.Acquire(ctx, r.GetWorkspaceId())
		if err != nil {
			return a.rpcError(err)
		}
		defer done()
		api = a.scoped(s, r.GetWorkspaceId())
	}
	return api.rpcError(api.Service.WatchFiles(ctx, r.GetPath(), func(refresh bool) error { return stream.Send(&v1.FileChange{Refresh: refresh}) }))
}

func (t *Transport) filesWatch(w http.ResponseWriter, r *http.Request) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	for key, values := range query {
		if (key != "path" && key != "workspace_id") || len(values) != 1 {
			err = fault.New(diag.InvalidArgument, diag.ServerPageInvalid)
		}
	}
	if err != nil || len(query.Get("path")) > 4096 || len(query.Get("workspace_id")) > 128 {
		t.writeControlReply(w, r, t.api.errorReply("", fault.New(diag.InvalidArgument, diag.ServerPageInvalid)))
		return
	}
	if id := query.Get("workspace_id"); id != "" {
		r.Header.Set("X-Meca-Workspace", id)
	}
	api, done, err := t.requestAPI(r)
	if err != nil {
		t.writeControlReply(w, r, t.api.errorReply("", err))
		return
	}
	defer done()
	select {
	case t.streams <- struct{}{}:
		defer func() { <-t.streams }()
	default:
		http.Error(w, diag.Text(diag.ServerapiTooManyObservers), http.StatusTooManyRequests)
		return
	}
	writer, err := eventstream.New(w)
	if err != nil {
		return
	}
	err = api.Service.WatchFiles(r.Context(), query.Get("path"), func(refresh bool) error {
		body, err := protojson.Marshal(&v1.FileChange{Refresh: refresh})
		if err != nil {
			return err
		}
		if err := writer.Send("change", "", body); err != nil {
			return err
		}
		return writer.Flush()
	})
	if err != nil && r.Context().Err() == nil {
		body, _ := protojson.Marshal(&v1.FileChange{Error: fault.PublicMessage(err, api.Redact)})
		_ = writer.Send("change", "", body)
		_ = writer.Flush()
	}
}
