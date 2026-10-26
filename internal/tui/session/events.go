// Purpose: Deliver worker results in order and apply them only in the session event loop.
// Pattern: Explicit event delivery and state transitions.
// Usage: Task and connection workers post notices; the terminal loop receives them.
// Related: internal/tui/session/app.go, internal/tui/session/tasks.go, internal/tui/session/model.go.
package session

import (
	"context"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	"github.com/keonho-kim/meca/internal/tools/local"
)

type notice struct {
	FileWatch       *watchResult
	Stream          *streamNotice
	FormRevision    int
	ModelForm       bool
	Diagnostic      bool
	History         *historyPage
	Directory       *directoryPage
	FileSelection   *fileSelectionResult
	SetupRevision   int
	Clipboard       bool
	RunStatus       runrecord.Status
	FilePreview     *local.FilePreview
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
		if a.worker.stopping() {
			return
		}
		a.proposal = n.Approval
		a.page = "approval"
		a.choices = []string{"Approve", "Reject"}
		a.choice = 0
		a.previewOffset = 0
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
