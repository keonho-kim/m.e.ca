// Purpose: Cache approval layout and move large preview indexing off the terminal loop.
// Pattern: Cancellable single-worker presentation state.
// Usage: renderMenu requests a layout; approval closure releases its source references.
// Related: internal/tui/session/foreground.go, internal/tui/view/preview.go, internal/tui/session/events.go.
package session

import (
	"context"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/view"
)

type approvalPreviewState struct {
	proposal *toolapi.Approval
	language locale.Language
	width    int
	layout   *view.Preview
	busy     bool
	job      uint64
	cancel   context.CancelFunc
}

type approvalPreviewNotice struct {
	proposal *toolapi.Approval
	language locale.Language
	width    int
	layout   *view.Preview
	job      uint64
}

func (a *app) releaseApprovalPreview() {
	p := &a.approvalPreview
	if p.cancel != nil {
		p.cancel()
	}
	p.proposal, p.layout = nil, nil
}

func (a *app) approvalPreviewFor(width int) *view.Preview {
	p := &a.approvalPreview
	if p.proposal == a.proposal && p.language == a.language && p.width == width && p.layout != nil {
		return p.layout
	}
	if p.busy {
		if p.cancel != nil && (p.proposal != a.proposal || p.language != a.language || p.width != width) {
			p.cancel()
		}
		return nil
	}
	source := view.ApprovalPreview(*a.proposal, a.language)
	p.proposal, p.language, p.width = a.proposal, a.language, width
	if len(source) <= 64<<10 {
		p.layout, _ = view.BuildPreview(a.ctx, source, width, a.proposal.Diff)
		return p.layout
	}
	ctx, cancel := context.WithCancel(a.ctx)
	p.busy, p.cancel = true, cancel
	p.job++
	value := approvalPreviewNotice{proposal: a.proposal, language: a.language, width: width, job: p.job}
	diff := a.proposal.Diff
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		value.layout, _ = view.BuildPreview(ctx, source, width, diff)
		_ = a.post(a.ctx, notice{ApprovalPreview: &value})
	}()
	return nil
}

func (a *app) receiveApprovalPreview(n *approvalPreviewNotice) {
	p := &a.approvalPreview
	if p.job != n.job {
		return
	}
	p.busy = false
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	if a.page != "approval" || a.proposal != n.proposal {
		return
	}
	p.proposal, p.language, p.width, p.layout = n.proposal, n.language, n.width, n.layout
}
