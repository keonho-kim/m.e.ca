// Purpose: Deliver worker results in order and apply them only in the session event loop.
// Pattern: Explicit event delivery and state transitions.
// Usage: Task and connection workers post notices; the terminal loop receives them.
// Related: internal/tui/session/app.go, internal/tui/session/tasks.go, internal/tui/session/model.go.
package session

import (
	"context"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/agent/run"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
	"github.com/keonho-kim/v-code/internal/tools/local"
)

type notice struct {
	FilePreview     *local.FilePreview
	PreviewRevision int
	RememberModel   bool
	credential      string
	Effort          *string
	Event           *run.Event
	Approval        *toolapi.Approval
	Done            bool
	Err             error
	Plan            *planning.Plan
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
	if n.PreviewRevision > 0 {
		a.receiveFilePreview(n)
		return
	}
	if n.Approval != nil {
		if a.stopRequested {
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
