// Purpose: Run model generation diagnostics only after an explicit user selection.
// Pattern: Cancellable presentation worker.
// Usage: !model offers the connection diagnostic separately from connecting and choosing effort.
// Related: internal/provider/diagnostic/check.go, internal/tui/session/menus.go.
package session

import (
	"context"
	"github.com/keonho-kim/meca/internal/provider/diagnostic"
)

func (a *app) diagnoseModel() {
	if a.busy || a.client == nil {
		a.status = a.language.Text("모델을 선택하고 현재 작업을 마친 뒤 진단하세요")
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.taskCancel = cancel
	a.busy = true
	a.startElapsed()
	if a.modelForm == nil {
		a.page = "chat"
	}
	a.status = a.language.Text("모델 연결 진단 중")
	a.setupRevision++
	revision := a.setupRevision
	client, selection, settings := a.client, a.selection, a.opts.Config.InferenceSettings()
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		nativeOn := selection.Details != nil && selection.Details.Reasoning != nil && selection.Details.Reasoning.Enabled()
		_, err := diagnostic.Check(ctx, client, selection.Model, selection.ThinkingSupported, nativeOn, &settings)
		if err == nil && selection.Effort() != "" {
			err = diagnostic.Effort(ctx, client, selection.Model, selection.Effort(), &settings)
		}
		a.post(a.ctx, notice{Done: true, SetupRevision: revision, Diagnostic: true, Err: err})
	}()
}
