// Purpose: Serve native gRPC, gRPC-Web and Connect from one generated service contract.
// Pattern: Protobuf RPC transport adapter.
// Usage: --protocol grpc mounts these handlers on an HTTP/2-capable server.
// Related: api/meca/v1/service.proto, internal/serverapi/watch.go.
package serverapi

import (
	"context"
	"net/http"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/serverapi/admission"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"github.com/keonho-kim/meca/internal/serverapi/gen/mecav1/mecav1connect"
	"google.golang.org/protobuf/proto"
)

func mountRPC(mux *http.ServeMux, api *API) {
	srv := connect.NewServer()
	mecav1connect.RegisterMecaServiceHandler(srv, api)
	connecthttp.Mount(mux, srv, connecthttp.WithReadMaxBytes(maxRequestBytes), connecthttp.WithSendMaxBytes(8<<20))
}

func (a *API) WatchEvents(ctx context.Context, request *v1.WatchRequest, stream mecav1connect.MecaServiceWatchEventsServerStream) error {
	if proto.Size(request) > 4096 {
		return connect.NewError(connect.CodeResourceExhausted, diagcatalog.Text(diagcatalog.ServerapiWatchRequestExceedsBytes))
	}
	admission.ReleaseInput(ctx)
	err := a.watch(ctx, request.GetCursor(), request.GetTaskId(), stream.Send)
	if err == nil {
		return nil
	}
	code := connect.CodeInternal
	switch diagcatalog.Semantic(faults.Code(err)) {
	case diagcatalog.Canceled:
		code = connect.CodeCanceled
	case diagcatalog.DeadlineExceeded:
		code = connect.CodeDeadlineExceeded
	case diagcatalog.InvalidArgument:
		code = connect.CodeInvalidArgument
	case diagcatalog.NotFound:
		code = connect.CodeNotFound
	case diagcatalog.PermissionDenied:
		code = connect.CodePermissionDenied
	case diagcatalog.ResourceExhausted:
		code = connect.CodeResourceExhausted
	case diagcatalog.Unavailable, diagcatalog.DeliveryFailed:
		code = connect.CodeUnavailable
	case diagcatalog.Conflict:
		code = connect.CodeAborted
	case diagcatalog.FailedPrecondition:
		code = connect.CodeFailedPrecondition
	}
	return connect.NewError(code, faults.PublicMessage(err, a.Redact))
}

func writeAuthError(w http.ResponseWriter, r *http.Request) {
	writer := connecthttp.NewErrorWriter()
	if writer.IsSupported(r) {
		_ = writer.Write(w, r, connect.NewError(connect.CodeUnauthenticated, diagcatalog.Text(diagcatalog.ServerapiBearerTokenIsRequired)))
	} else {
		http.Error(w, diagcatalog.Text(diagcatalog.ServerapiBearerTokenIsRequired), http.StatusUnauthorized)
	}
}
