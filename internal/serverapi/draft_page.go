// Purpose: Mask draft pages with boundary lookaround and pin reads to one credential revision.
// Pattern: Versioned public read boundary.
// Usage: Native draft commands return source byte offsets without exposing adjacent text.
// Related: internal/commands/server/draft_original.go, internal/redaction/range.go.
package serverapi

import (
	"context"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func (a *API) draftPage(ctx context.Context, r *v1.DraftRequest) (*v1.DraftPage, error) {
	if a.RedactRange == nil || a.RedactionVersion == nil {
		return nil, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerWorkspaceUnavailable)
	}
	if r.GetOffset() > 0 && r.Revision == nil {
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
	}
	version := int64(a.RedactionVersion())
	if r.Revision != nil && r.GetRevision() != version {
		return nil, faults.New(diagcatalog.Conflict, diagcatalog.ServerReadRevision)
	}
	limit := int(r.GetLimit())
	if limit == 0 {
		limit = 64 << 10
	}
	page, err := a.Service.ReadDraft(ctx, r.GetTaskId(), r.GetAgent(), r.GetGeneration(), r.GetOffset(), limit, r.GetThinking())
	if err != nil {
		return nil, err
	}
	text, err := a.RedactRange(page.Window, page.Start, page.End)
	if err != nil {
		return nil, err
	}
	if int64(a.RedactionVersion()) != version {
		return nil, faults.New(diagcatalog.Conflict, diagcatalog.ServerReadRevision)
	}
	return &v1.DraftPage{Text: text, Generation: page.Generation, Offset: page.Offset, NextOffset: page.Next, TotalBytes: page.Total, Truncated: page.Truncated, Revision: version}, nil
}
