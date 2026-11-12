// Purpose: Persist global terminal settings and prepare localized history outside the input loop.
// Pattern: Joined settings worker with event-loop-only adoption.
// Usage: Confirmed language and permission saves share the existing foreground worker lifetime.
// Related: internal/tui/session/events.go, internal/tui/session/language.go, internal/tui/session/history.go.
package session

import (
	"context"
	"errors"

	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

type settingsResult struct {
	page     string
	config   *config.Config
	language locale.Language
	history  *historyPage
	err      error
}

func (a *app) startSettingsSave(page string, save func(context.Context) (config.Config, error)) {
	if a.work.state.active() {
		a.status = a.language.Text("Stop the current task before changing settings")
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.work.cancel, a.work.state = cancel, workerSetup
	a.status = ""
	a.startElapsed()
	store, id, redact := a.opts.Store, a.opts.Session.ID, a.opts.Vault.Redact
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		result := settingsResult{page: page}
		next, err := save(ctx)
		result.err = err
		if err == nil {
			// A committed preference is adopted even if cancellation or a later
			// history read fails. Retrying must not pretend the write never occurred.
			result.config = &next
			if page == "language" {
				result.language = locale.Resolve(ctx, next.Language)
				result.history, result.err = readLocalizedHistory(ctx, store, id, result.language, redact)
			}
		}
		_ = a.post(a.ctx, notice{Settings: &result})
	}()
}

func (a *app) receiveSettings(result *settingsResult) {
	returnToChat := a.page == result.page || a.page == "chat"
	a.work.state, a.work.cancel = workerIdle, nil
	a.stopElapsed()
	if result.config != nil {
		a.opts.Config = *result.config
		a.settings.Permission = result.config.Permission
		if result.page == "language" {
			a.language = result.language
			a.clearSelection()
			a.chat.chatProjection = chatProjection{}
			a.scrollbar = scrollbarState{}
			if result.history != nil {
				a.applyHistory(result.history)
			}
		}
		if returnToChat {
			a.page = "chat"
			a.menuInput.Set("")
			a.stopNotice, a.status = "", ""
		}
	}
	if result.err != nil && !errors.Is(result.err, context.Canceled) {
		a.fail(result.err)
	}
}
