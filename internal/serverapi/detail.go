// Purpose: Bind an authorized detail view to the API-owned cache and UTF-8 byte pages.
// Pattern: Detail wire boundary adapter.
// Usage: API.Call dispatches detail requests here; callers receive the selected public projection.
// Related: internal/commands/server/detail.go, internal/serverapi/detail/cache.go.
package serverapi

import (
	"context"
	"unicode/utf8"

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
	if r.GetView() < v1.DetailRequest_DOCUMENT || r.GetView() > v1.DetailRequest_THINKING {
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerChooseExactlyOneValidDetailReference)
	}
	read, err := a.Service.OpenDetail(ctx, server.DetailSource{View: server.DetailView(r.GetView()), Session: r.GetSessionId(), Tool: record.EventReference{Session: r.GetTool().GetSessionId(), Sequence: r.GetTool().GetSequence()}, ApprovalID: r.GetApprovalId(), HistorySequence: r.GetHistorySequence()})
	if err != nil {
		return nil, err
	}
	defer read.Release()
	data, total, revision, err := a.Details.Read(read.Context, detail.Request{Expected: r.GetRevision(), Session: read.Session, Source: read.Source, Version: a.RedactionVersion, NewMask: a.NewMask, Load: read.Load}, r.GetOffset(), limit)
	if err != nil {
		return nil, err
	}
	end := len(data)
	for cut := 0; cut <= 3 && cut <= len(data); cut++ {
		if utf8.Valid(data[:len(data)-cut]) {
			end = len(data) - cut
			break
		}
	}
	if !utf8.Valid(data[:end]) || (end == 0 && len(data) > 0) {
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
	}
	return &v1.DetailPage{Text: string(data[:end]), Offset: r.GetOffset(), NextOffset: r.GetOffset() + int64(end), TotalBytes: total, ContentType: detailContentType(r.GetView()), Revision: revision}, nil
}

func detailContentType(view v1.DetailRequest_View) string {
	if view == v1.DetailRequest_TEXT || view == v1.DetailRequest_THINKING {
		return "text/plain; charset=utf-8"
	}
	return "application/json; charset=utf-8"
}
