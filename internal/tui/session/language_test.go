// Purpose: Verify language selection updates visible UI and persisted preferences while preserving conversation literals.
// Pattern: Virtual terminal settings tests.
// Usage: go test ./internal/tui/session -run TestLanguage.
// Related: internal/tui/session/language.go, internal/tui/session/history_load.go, internal/config/language.go.
package session

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func TestLanguageMenuAppliesAndPersists(t *testing.T) {
	a := fixture(t)
	id := a.opts.Session.ID
	for _, message := range []modelapi.Message{{Role: "user", Content: "Keep 한글 INPUT literal"}, {Role: "assistant", Content: "Original 한국어 answer"}} {
		if err := a.opts.Store.AppendMessage(a.ctx, id, message); err != nil {
			t.Fatal(err)
		}
	}
	cp := run.Checkpoint{ID: "run_language", Objective: "test", Status: run.Answered, Phase: "finished", InstructionsDigest: strings.Repeat("a", 64)}
	stats := run.Statistics{ElapsedMillis: 1000, GenerationMillis: 1000, Usage: &modelapi.TokenUsage{Input: 2, Output: 1}}
	if err := a.opts.Store.RecordEvent(a.ctx, id, run.Event{Kind: "run_checkpoint", RunID: cp.ID, Checkpoint: &cp, Statistics: &stats}); err != nil {
		t.Fatal(err)
	}
	before, err := a.opts.Store.Messages(a.ctx, id, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.loadHistory(); err != nil {
		t.Fatal(err)
	}
	a.output.Collapsed = true
	typeInput(a, "!language")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.render()
	if a.page != "language" || !strings.Contains(screenLine(a.screen, 2, 80), "시스템 기본값") || keymapLabels(t, a)["Enter"] != "저장 확인" {
		t.Fatal("language menu or keymap missing")
	}
	a.editor.Set("preserved draft")
	for range 2 {
		a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
	if a.page != "chat" || a.language != locale.English || a.opts.Config.Language != config.LanguageEnglish || a.editor.String() != "preserved draft" || !a.output.Collapsed {
		t.Fatal("language did not apply without changing unrelated state")
	}
	a.render()
	if !strings.HasPrefix(screenLine(a.screen, 0, 80), "STATUS IDLE") || !strings.Contains(a.output.Blocks[0].Text, "\n\nAgent") || !strings.Contains(a.output.Blocks[2].Text, "Worked for 00:00:01 | INPUT 2 | OUTPUT 1 | TPS 1.0") {
		t.Fatal("existing transcript labels were not refreshed")
	}
	after, err := a.opts.Store.Messages(a.ctx, id, 10)
	if err != nil || !reflect.DeepEqual(before, after) || !strings.Contains(a.output.Blocks[0].Text, before[0].Content) || !strings.Contains(a.output.Blocks[1].Text, before[1].Content) {
		t.Fatal("language change modified conversation literals")
	}
	saved, err := config.Load(a.opts.State)
	if err != nil || saved.Language != config.LanguageEnglish {
		t.Fatalf("saved language=%v err=%v", saved.Language, err)
	}
	a.openLanguage()
	if a.choice != 2 {
		t.Fatal("current preference was not selected")
	}
}

func TestLanguageSessionStartup(t *testing.T) {
	for _, test := range []struct {
		preference config.Language
		host, want string
	}{{config.LanguageKorean, "C", "상태 대기"}, {config.LanguageEnglish, "ko_KR.UTF-8", "STATUS IDLE"}} {
		t.Run(string(test.preference), func(t *testing.T) {
			t.Setenv("LC_ALL", test.host)
			a := fixture(t)
			a.opts.Config.Language = test.preference
			ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
			done := make(chan struct{})
			go func() {
				defer close(done)
				for _, event := range []*tcell.EventKey{tcell.NewEventKey(tcell.KeyRune, "!quit", 0), tcell.NewEventKey(tcell.KeyEnter, "", 0)} {
					select {
					case a.screen.EventQ() <- event:
					case <-ctx.Done():
						return
					}
				}
			}()
			_, err := Run(ctx, a.screen, a.opts)
			cancel()
			<-done
			if err != nil || !strings.HasPrefix(screenLine(a.screen, 0, 80), test.want) {
				t.Fatalf("session language not applied: %q, %v", screenLine(a.screen, 0, 80), err)
			}
		})
	}
}

func TestLanguageFailureAndCancelPreserveSelection(t *testing.T) {
	a := fixture(t)
	a.language = locale.English
	a.output.Append("preserved output", false)
	a.opts.SaveConfig = func(config.Config) error { return errors.New("config write failed") }
	a.openLanguage()
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
	if a.page != "language" || a.language != locale.English || a.opts.Config.Language != config.LanguageAuto || !strings.Contains(a.status, "config write failed") || a.output.Blocks[0].Text != "preserved output" {
		t.Fatal("failed save changed the active language or history")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "chat" || a.language != locale.English || a.opts.Config.Language != config.LanguageAuto {
		t.Fatal("cancelling language selection changed its value")
	}
}

func TestLanguageReloadDiscardsOldHistoryResults(t *testing.T) {
	a := fixture(t)
	a.historyRevision = 3
	a.historyLoading = true
	a.receiveHistory(&historyPage{revision: 2, blocks: []viewport.Block{{Text: "old language"}}})
	if len(a.output.Blocks) != 0 || !a.historyLoading {
		t.Fatal("stale history replaced the refreshed language or current loading state")
	}
	a.receiveHistory(&historyPage{revision: 3, cursor: 1, blocks: []viewport.Block{{Text: "current language"}}})
	if a.historyLoading || len(a.output.Blocks) != 1 || a.output.Blocks[0].Text != "current language" {
		t.Fatal("current history result was discarded")
	}
}
