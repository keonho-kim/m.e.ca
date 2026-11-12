// Purpose: Keep keyboard handling responsive while global settings wait for persistence.
// Pattern: Controlled I/O boundary tests with virtual scheduling.
// Usage: go test ./internal/tui/session -run TestSettingsSave.
// Related: internal/tui/session/settings_save.go, internal/tui/session/language.go, internal/tui/session/permission.go.
package session

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/storage/history"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

type settingsHistoryStore struct{ Store }

func (*settingsHistoryStore) Timeline(context.Context, string, int64, int) ([]history.Record, int64, error) {
	return nil, 0, nil
}

type unreadableSettingsHistory struct{ Store }

func (*unreadableSettingsHistory) Timeline(context.Context, string, int64, int) ([]history.Record, int64, error) {
	return nil, 0, errors.New("history read failed")
}

func TestSettingsSavePreservesCommitWhenHistoryFails(t *testing.T) {
	a := screenFixture(t)
	a.opts.Store = &unreadableSettingsHistory{}
	a.chat.output.Append("preserved conversation", false)
	a.choice = 2
	next := a.opts.Config
	a.opts.UpdateConfig = func(_ context.Context, apply func(*config.Config)) (config.Config, error) {
		apply(&next)
		return next, nil
	}
	a.saveLanguage()
	a.receive(<-a.events)
	if a.opts.Config.Language != config.LanguageEnglish || a.work.state.active() || a.chat.output.Blocks[0].Text != "preserved conversation" || a.status == "" {
		t.Fatal("history failure lost an already committed preference or conversation")
	}
}

func TestSettingsSaveDoesNotStealForegroundAfterNavigation(t *testing.T) {
	a := screenFixture(t)
	a.page = "files"
	a.work.state = workerSetup
	a.menuInput.Set("keep current navigation")
	next := a.opts.Config
	next.Language = config.LanguageEnglish
	a.receiveSettings(&settingsResult{page: "language", config: &next, language: locale.English, history: &historyPage{}})
	if a.page != "files" || a.menuInput.String() != "keep current navigation" || a.opts.Config.Language != config.LanguageEnglish {
		t.Fatal("an asynchronous settings result replaced newer foreground input")
	}
}

func TestSettingsSaveDoesNotBlockInput(t *testing.T) {
	for _, page := range []string{"language", "permission"} {
		t.Run(page, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				a := screenFixture(t)
				a.opts.Store = &settingsHistoryStore{}
				a.page = page
				a.choice = 1
				release := make(chan struct{})
				finish := sync.OnceFunc(func() { close(release) })
				defer finish()
				current := a.opts.Config
				a.opts.UpdateConfig = func(ctx context.Context, apply func(*config.Config)) (config.Config, error) {
					select {
					case <-release:
					case <-ctx.Done():
						return current, ctx.Err()
					}
					next := current
					apply(&next)
					return next, nil
				}
				returned := make(chan struct{})
				go func() {
					if page == "language" {
						a.saveLanguage()
					} else {
						a.savePermission()
					}
					close(returned)
				}()
				synctest.Wait()
				select {
				case <-returned:
				default:
					t.Fatal("settings persistence blocked its event-loop caller")
				}
				a.compose.editor.Insert("draft remains responsive")
				finish()
				n := <-a.events
				a.receive(n)
				if a.work.state.active() || a.page != "chat" || a.compose.editor.String() != "draft remains responsive" {
					t.Fatal("settings adoption lost input or worker ownership")
				}
			})
		})
	}
}
