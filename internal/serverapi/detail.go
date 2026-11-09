// Purpose: Bind a validated conversation read lease to the API-owned JSON document cache and byte pages.
// Pattern: Detail wire boundary adapter.
// Usage: API.Call dispatches detail requests here; callers receive only the stable public JSON schema.
// Related: internal/commands/server/detail.go, internal/serverapi/detail/cache.go.
package serverapi

import (
	"context"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/server"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/serverapi/detail"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func (a *API) readDetail(ctx context.Context, r *v1.DetailRequest) (*v1.DetailPage, error) {
	limit := int(r.GetLimit())
	if limit == 0 {
		limit = 64 << 10
	}
	if r.GetOffset() < 0 || r.GetOffset() > 0 && r.GetRevision() == "" || len(r.GetRevision()) > 64 || limit < 1 || limit > 64<<10 {
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.DetailDetailOffsetMustBeNonnegativeAndLimit)
	}
	read, err := a.Service.OpenDetail(ctx, server.DetailSource{Session: r.GetSessionId(), Tool: record.EventReference{Session: r.GetTool().GetSessionId(), Sequence: r.GetTool().GetSequence()}, ApprovalID: r.GetApprovalId(), HistorySequence: r.GetHistorySequence()})
	if err != nil {
		return nil, err
	}
	defer read.Release()
	data, total, revision, err := a.Details.Read(read.Context, detail.Request{Expected: r.GetRevision(), Session: read.Session, Source: read.Source, Version: a.RedactionVersion, NewMask: a.NewMask, Load: read.Load}, r.GetOffset(), limit)
	if err != nil {
		return nil, err
	}
	return &v1.DetailPage{Data: data, Offset: r.GetOffset(), NextOffset: r.GetOffset() + int64(len(data)), TotalBytes: total, ContentType: "application/json; charset=utf-8", Revision: revision}, nil
}
