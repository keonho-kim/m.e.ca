// Purpose: Deliver worker results in order and apply them only in the session event loop.
// Pattern: Explicit event delivery and state transitions.
// Usage: Agent and connection workers post notices; the terminal loop receives them.
// Related: internal/tui/session/app.go, internal/tui/session/tasks.go, internal/tui/session/model.go.
package session

import (
	"context"

	"github.com/keonho-kim/meca/internal/commands/modelchange"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/tui/browser/controller"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/fileview"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
)

type notice struct {
	ImagePaste      *imagePasteNotice
	ApprovalPreview *approvalPreviewNotice
	Skills          *skillNotice
	Commit          *modelchange.Committed
	Config          *config.Config
	Detail          *detailResult
	PreviewLayout   *controller.LayoutResult
	FileWatch       *controller.WatchResult
	Stream          *streamNotice
	FormRevision    int
	ModelForm       bool
	Diagnostic      bool
	History         *historyPage
	Directory       *controller.DirectoryPage
	FileSelection   *fileSelectionResult
	SetupRevision   int
	Clipboard       bool
	RunStatus       runrecord.Status
	FilePreview     *fileview.Preview
	PreviewRevision int
	RememberModel   bool
	credential      string
	Event           *runrecord.Event
	Approval        *toolapi.Approval
	Done            bool
	Err             error
	Client          *httpclient.Client
	Models          []string
	Selection       *modelconn.Selection
}

func (a *app) post(ctx context.Context, n notice) error {
	select {
	case a.events <- n:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (a *app) receive(n notice) {
	if n.ImagePaste != nil {
		a.receiveImagePaste(n.ImagePaste)
		return
	}
	if n.ApprovalPreview != nil {
		a.receiveApprovalPreview(n.ApprovalPreview)
		return
	}
	if n.Skills != nil {
		a.receiveSkills(n.Skills)
		return
	}
	if n.Detail != nil {
		a.receiveToolDetail(n.Detail)
		return
	}
	if n.PreviewLayout != nil {
		a.receivePreviewLayout(n.PreviewLayout)
		return
	}
	if n.FileWatch != nil {
		a.receiveFileWatch(n.FileWatch)
		return
	}
	if n.Stream != nil {
		if event := n.Stream.take(); event != nil {
			a.receiveRuntime(event)
		}
		return
	}
	if n.History != nil {
		a.receiveHistory(n.History)
		return
	}
	if n.Directory != nil {
		a.receiveDirectory(n.Directory)
		return
	}
	if n.FileSelection != nil {
		a.receiveFileSelection(n.FileSelection)
		return
	}
	if !a.setupResult(&n) {
		return
	}
	if n.Clipboard {
		a.clipboardBusy = false
		if n.Err != nil {
			a.stopNotice = a.language.Text("클립보드 복사 실패: ") + a.opts.Vault.Redact(n.Err.Error())
		} else {
			a.stopNotice = a.language.Text("복사했습니다")
		}
		return
	}
	if n.PreviewRevision > 0 {
		a.receiveFilePreview(n)
		return
	}
	if n.Approval != nil {
		if a.work.state.stopping() {
			return
		}
		a.openApproval(n.Approval)
		return
	}
	if n.Event != nil {
		a.receiveRuntime(n.Event)
		return
	}
	if n.Done {
		a.finishWorker(n)
	}
}
