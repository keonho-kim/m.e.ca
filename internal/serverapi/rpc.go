// Purpose: Serve native gRPC, gRPC-Web and Connect from one generated service contract.
// Pattern: Protobuf RPC transport adapter.
// Usage: --protocol grpc mounts these handlers on an HTTP/2-capable server.
// Related: api/meca/v1/service.proto, internal/serverapi/watch.go.
package serverapi

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/keonho-kim/meca/internal/commands/server"
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
		return connect.NewError(connect.CodeResourceExhausted, "watch request exceeds 4096 bytes")
	}
	admission.ReleaseInput(ctx)
	err := a.watch(ctx, request.GetCursor(), stream.Send)
	if errors.Is(err, context.Canceled) {
		return connect.NewError(connect.CodeCanceled, "observer disconnected")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return connect.NewError(connect.CodeDeadlineExceeded, "observer deadline exceeded")
	}
	var domain *server.Error
	if errors.As(err, &domain) && domain.Code == "invalid_argument" {
		return connect.NewError(connect.CodeInvalidArgument, domain.Message)
	}
	return connect.NewError(connect.CodeUnavailable, "event observer closed")
}

func writeAuthError(w http.ResponseWriter, r *http.Request) {
	writer := connecthttp.NewErrorWriter()
	if writer.IsSupported(r) {
		_ = writer.Write(w, r, connect.NewError(connect.CodeUnauthenticated, "bearer token is required"))
	} else {
		http.Error(w, "bearer token is required", http.StatusUnauthorized)
	}
}
