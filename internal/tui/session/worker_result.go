// Purpose: Adopt completed worker results and restore the terminal interaction state.
// Pattern: Worker result state transition.
// Usage: The event loop invokes finishWorker when a task or connection completes.
// Related: internal/tui/session/events.go, internal/tui/session/connection.go.
package session

import (
	"context"
	"errors"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/transcript"
)

func (a *app) finishWorker(n notice) {
	a.restoreSteering()
	active, cancelled := a.work.state.agent(), a.work.state.stopping()
	defer func() {
		if active {
			a.finishPending(n.Err == nil && !cancelled && n.RunStatus == runrecord.Answered)
		}
	}()
	a.chat.tokenCurrent = modelapi.TokenUsage{}
	stopped := a.work.state.stopping()
	a.work.state = workerIdle
	a.stopCompression()
	a.stopElapsed()
	a.clearToolActivity()
	a.work.cancel = nil
	if n.Err != nil {
		a.chat.output.AppendBlock(transcript.Thought(a.chat.thinking))
		a.chat.output.AppendBlock(transcript.Answer(a.chat.stream))
	}
	if active && a.chat.runSummary != nil {
		a.chat.output.AppendBlock(*a.chat.runSummary)
	}
	a.chat.runSummary = nil
	a.chat.stream, a.chat.thinking = "", ""
	a.status = ""
	if n.Err != nil && !(stopped && errors.Is(n.Err, context.Canceled)) {
		a.status = a.opts.Vault.Redact(n.Err.Error())
	}
	if stopped && errors.Is(n.Err, context.Canceled) && a.page == "chat" {
		a.stopNotice = a.language.Text("중지했습니다. 다음 입력으로 이어갈 수 있습니다")
		if n.SetupRevision > 0 {
			a.stopNotice = a.language.Text("연결 작업을 취소했습니다")
		}
	}
	if n.ModelForm {
		a.finishModelFormSetup(n)
		return
	}
	if _, err := a.adoptConnection(n); err != nil {
		n.Err = err
		a.status = a.opts.Vault.Redact(err.Error())
	}
	if n.Diagnostic && n.Err == nil {
		a.status = a.language.Text("모델 연결 진단을 통과했습니다")
	}
	if n.Diagnostic && a.connection.modelForm != nil && a.connection.modelForm.Saving {
		if n.Err != nil {
			a.connection.modelForm.Saving, a.page = false, "model"
		} else {
			a.saveModelForm()
		}
	}
	if a.proposal != nil {
		a.closeApproval()
	}
}
