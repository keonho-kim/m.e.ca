// Purpose: Adopt completed worker results and restore the terminal interaction state.
// Pattern: Worker result state transition.
// Usage: The event loop invokes finishWorker when a task or connection completes.
// Related: internal/tui/session/events.go, internal/tui/session/connection.go.
package session

import (
	"context"
	"errors"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/tui/transcript"
)

func (a *app) finishWorker(n notice) {
	active, cancelled := a.agentActive, a.stopRequested
	defer func() {
		if active {
			a.finishPending(n.Err == nil && !cancelled && n.RunStatus == run.Answered)
		}
	}()
	a.tokenCurrent = modelapi.TokenUsage{}
	if n.Err == nil && n.Effort != nil {
		n.Err = a.applyEffort(*n.Effort)
	}
	stopped := a.stopRequested
	a.stopRequested = false
	a.agentActive = false
	a.stopCompression()
	a.stopElapsed()
	a.busy = false
	a.taskCancel = nil
	if n.Err != nil {
		a.output.Append(a.thinking, true)
		a.output.AppendBlock(transcript.Answer(a.stream))
	}
	a.stream, a.thinking = "", ""
	a.status = ""
	if n.Err != nil && !(stopped && errors.Is(n.Err, context.Canceled)) {
		a.status = a.opts.Vault.Redact(n.Err.Error())
	}
	if stopped && errors.Is(n.Err, context.Canceled) && a.page == "chat" {
		a.stopNotice = "중지했습니다. 다음 입력으로 이어갈 수 있습니다"
		if n.SetupRevision > 0 {
			a.stopNotice = "연결 작업을 취소했습니다"
		}
	}
	if err := a.adoptConnection(n); err != nil {
		n.Err = err
		a.status = a.opts.Vault.Redact(err.Error())
	}
	if n.Models != nil && !errors.Is(n.Err, context.Canceled) && !errors.Is(n.Err, context.DeadlineExceeded) {
		a.openModelInput(n.Models)
		if n.Err != nil {
			a.status = "목록 조회 실패 · 모델 ID를 직접 입력하세요: " + a.opts.Vault.Redact(n.Err.Error())
		}
	}
	if a.proposal != nil {
		a.proposal = nil
		a.page = "chat"
	}
}
